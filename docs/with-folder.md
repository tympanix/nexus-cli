# Feature: --with-folder — download the folder containing the first matched file

Summary
- Add a simple flag --with-folder that, when used with the existing repository/path source syntax, finds the first asset matching the provided path (matching is performed by Nexus) and downloads every file in that asset's containing folder.
- The folder download is performed recursively by the CLI — you do not need to pass --recursive when using --with-folder.

Flag
- --with-folder (bool) — when provided, locate the first matching asset and download all assets in its folder (recursively).

Source format and matching rules
- The source argument keeps the existing syntax: repository/path
- The CLI forwards the path portion directly to the Nexus Search API using the existing "name" query argument. The CLI does not perform client-side pattern validation or transformation.
- Pattern semantics and support are determined solely by the Nexus server. Quote patterns on the shell to avoid expansion (e.g. "my-repo/path/to/manifest-*.csv").

Behavior
- When --with-folder is provided:
  1. The CLI issues a Nexus search request using the provided repository and the src value as the "name" parameter.
  2. The CLI takes the first asset returned by Nexus (no client-side reordering/filtering).
  3. It derives the asset's containing folder (path.Dir(asset.Path)).
  4. It downloads all assets under that folder recursively using the CLI's existing download logic (preserves relative paths and respects flags such as --flatten, checksum/force, compression handling, etc.). There is no need to pass --recursive — recursion is implied for the folder download.
- If no assets are returned by Nexus for the provided name value, the command exits non-zero and reports that no matches were found.

Examples
- Exact filename (download folder containing manifest.csv):
  nexuscli-go download my-repo/path/to/manifest.csv ./local --with-folder

- Wildcard in filename (server-side semantics apply):
  nexuscli-go download "my-repo/path/to/manifest-*.csv" ./local --with-folder

Notes
- This feature forwards the src value to Nexus unchanged; pattern interpretation and matching behavior depend on the Nexus server version and configuration.
- The implementation intentionally performs no client-side pattern matching or normalization for this feature.
- --with-folder affects only the first match returned by Nexus (deterministic according to Nexus' response). There is no option in this feature to download folders for multiple matches.
- For large repositories prefer patterns with a fixed path prefix so the server-side search is efficient.

Important: how other flags interact
- The initial search request is performed solely using the src value (sent as the "name" parameter) and is not affected by client-side download filters.
- Download-level flags that filter or modify which files are saved (for example: --glob) apply only to the subsequent recursive download of the containing folder. In other words, the Nexus search that selects the anchor file is always performed using the raw src value; options like --glob influence which files from the folder are actually downloaded during the recursive fetch that follows.
