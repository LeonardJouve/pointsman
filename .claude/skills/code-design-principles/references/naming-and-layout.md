# Names and Aesthetics

## Contents
- Names
- Aesthetics

## Names [2][1]

| Rule | Bad | Better |
|---|---|---|
| Use specific verbs. `get` hides whether the page comes from a cache or the web. | `GetPage(url)` | `FetchPage(url)`, `DownloadPage(url)` |
| Avoid generic names; name what the value is. | `retval += v[i];` | `sum_squares += v[i];` |
| Put units and attributes in the name. | `delay`, `password`, `html` | `delay_secs`, `plaintext_password`, `html_utf8` |
| Longer scope, more precise name. A short name is fine for a three-line scope. | `days` (field) | `days_since_last_update` |
| Do not abbreviate beyond what a new teammate decodes at once. | `BEManager` | `BackEndManager` |
| Drop words that add nothing. | `convert_to_string` | `to_string` |
| Inclusive limits: `max_` / `min_`. | `CART_TOO_BIG_LIMIT = 10` | `MAX_ITEMS_IN_CART = 10` |
| Inclusive range: `first` / `last`. Half-open range: `begin` / `end`. | `start`, `stop` | `first`, `last` |
| Booleans read as questions: `is`, `has`, `can`, `should`. Avoid negated names. | `read_password`, `disable_ssl` | `need_password`, `use_ssl` |
| A name must not suggest something false. `filter` can mean keep or drop. | `results.filter("year <= 2011")` | `results.select(...)` or `results.exclude(...)` |
| Cheap-looking names must be cheap. | `getMean()` that loops over a million rows | `computeMean()` |

Test for any name: *what else could this mean?* Ask a teammate. If the first guess is wrong, rename.

Also watch for:
- Vague Name: the name conveys little [1].
- Hard to Pick Name: no precise name fits, which usually means the entity does two jobs [1].

## Aesthetics [2]

1. Be consistent: similar code has a similar shape.
2. Group related members under a one-line label:

```cpp
class FrontendServer {
    public:
        FrontendServer();
        ~FrontendServer();

        // Handlers
        void ViewProfile(HttpRequest* request);
        void SaveProfile(HttpRequest* request);

        // Request/Reply Utilities
        string ExtractQueryParam(HttpRequest* request, string param);
        void ReplyOK(HttpRequest* request, string html);
        void ReplyNotFound(HttpRequest* request, string error);

        // Database Helpers
        void OpenDatabase(string location, string user);
        void CloseDatabase(string location);
};
```

3. Remove repeated shape with a helper so that only the data differs, and align columns so differences stand out:

```cpp
CheckFullName("Doug Adams",   "Mr. Douglas Adams", "");
CheckFullName(" Jake Brown ", "Mr. Jake Brown III", "");
CheckFullName("No Such Guy",  "",                  "no match found");
CheckFullName("John",         "",                  "more than one result");
```

4. Split a multi-step function into paragraphs separated by blank lines, each with an optional one-line summary.
5. Choose one meaningful order (matching a form, importance, alphabetical) and keep it everywhere the same items appear.
