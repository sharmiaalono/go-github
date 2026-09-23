import unittest
import urllib.parse

from github_pagination import ListOptions, add_options, paginate_items, parse_pagination


class PaginationTests(unittest.TestCase):
    def test_add_options_handles_default_per_page(self):
        base_url = "https://api.example.com/repos"
        url_default = add_options(base_url, ListOptions(page=1, per_page=0))
        parsed_default = urllib.parse.urlparse(url_default)
        qs_default = urllib.parse.parse_qs(parsed_default.query)
        self.assertEqual(qs_default.get("page"), ["1"])
        self.assertNotIn("per_page", qs_default)

        url_custom = add_options(base_url, ListOptions(page=2, per_page=50))
        parsed_custom = urllib.parse.urlparse(url_custom)
        qs_custom = urllib.parse.parse_qs(parsed_custom.query)
        self.assertEqual(qs_custom.get("page"), ["2"])
        self.assertEqual(qs_custom.get("per_page"), ["50"])

    def test_parse_pagination_extracts_pages_with_or_without_per_page(self):
        link_header = (
            '<https://api.example.com/resource?page=2&per_page=30>; rel="next", '
            '<https://api.example.com/resource?page=1&per_page=30>; rel="prev", '
            '<https://api.example.com/resource?page=5&per_page=30>; rel="last", '
            '<https://api.example.com/resource?page=1&per_page=30>; rel="first"'
        )
        resp = parse_pagination(link_header)
        self.assertEqual(resp.NextPage, 2)
        self.assertEqual(resp.PrevPage, 1)
        self.assertEqual(resp.FirstPage, 1)
        self.assertEqual(resp.LastPage, 5)

        # Missing per_page parameter should not affect page parsing.
        link_header_no_per_page = (
            '<https://api.example.com/resource?page=3>; rel="next", '
            '<https://api.example.com/resource?page=2>; rel="prev"'
        )
        resp2 = parse_pagination(link_header_no_per_page)
        self.assertEqual(resp2.NextPage, 3)
        self.assertEqual(resp2.PrevPage, 2)

    def test_multi_page_flow_respects_default_per_page(self):
        items = list(range(65))  # 3 pages worth of data when default per_page=30
        opts = ListOptions(page=1, per_page=0)
        base_url = "https://api.example.com/issues"

        # First page
        page1_items, link1 = paginate_items(items, opts, base_url)
        self.assertEqual(len(page1_items), 30)
        resp1 = parse_pagination(link1)
        self.assertEqual(resp1.NextPage, 2)
        # Keep per_page at zero to mimic omitting it in subsequent requests
        opts.page = resp1.NextPage

        # Second page
        page2_items, link2 = paginate_items(items, opts, base_url)
        self.assertEqual(len(page2_items), 30)
        resp2 = parse_pagination(link2)
        self.assertEqual(resp2.NextPage, 3)
        opts.page = resp2.NextPage

        # Final page
        page3_items, link3 = paginate_items(items, opts, base_url)
        self.assertEqual(len(page3_items), 5)
        resp3 = parse_pagination(link3)
        self.assertEqual(resp3.NextPage, 0)
        self.assertEqual(page3_items, items[60:])


if __name__ == "__main__":
    unittest.main()
