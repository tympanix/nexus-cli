import os
import sys
import requests
from typing import Optional
from tqdm import tqdm
import concurrent.futures

NEXUS_URL = os.environ.get("NEXUS_URL", "http://localhost:8081")
USERNAME = os.environ.get("NEXUS_USER", "admin")
PASSWORD = os.environ.get("NEXUS_PASS", "admin")

SEARCH_ENDPOINT = f"{NEXUS_URL}/service/rest/v1/search/assets"

def list_assets(repository: str, src: str) -> list:
    continuation_token = None
    assets = []
    params = {
        'repository': repository,
        'format': 'raw',
        'direction': 'asc',
        'sort': 'name',
        'q': f"/{src}/*"
    }
    while True:
        if continuation_token:
            params['continuationToken'] = continuation_token
        response = requests.get(SEARCH_ENDPOINT, params=params, auth=(USERNAME, PASSWORD))
        if response.status_code != 200:
            raise Exception(f"Failed to list assets: {response.status_code} {response.text}")
        data = response.json()
        for item in data.get('items', []):
            assets.append(item)
        continuation_token = data.get('continuationToken')
        if not continuation_token:
            break
    return assets

def download_asset(asset: dict, dest_dir: str, quiet: bool = False):
    download_url = asset['downloadUrl']
    path = asset['path'].lstrip("/")
    local_path = os.path.join(dest_dir, path)
    os.makedirs(os.path.dirname(local_path), exist_ok=True)
    
    # Show progress bar only if stdout is a TTY and not in quiet mode
    show_progress = sys.stdout.isatty() and not quiet
    
    with requests.get(download_url, auth=(USERNAME, PASSWORD), stream=True) as r:
        r.raise_for_status()
        total = int(r.headers.get('content-length', 0))
        with open(local_path, 'wb') as f, tqdm(
            desc=f"Downloading {path}",
            total=total,
            unit='B',
            unit_scale=True,
            disable=not show_progress
        ) as bar:
            for chunk in r.iter_content(chunk_size=8192):
                if chunk:
                    f.write(chunk)
                    bar.update(len(chunk))

def download_folder(src_arg: str, dest_dir: str, quiet: bool = False) -> bool:
    if '/' not in src_arg:
        if not quiet:
            print("Error: The src argument must be in the form 'repository/folder' or 'repository/folder/subfolder'.")
        return False
    repository, src = src_arg.split('/', 1)
    assets = list_assets(repository, src)
    if not assets:
        if not quiet:
            print(f"No assets found in folder '{src}' in repository '{repository}'")
        return True
    max_workers = min(8, len(assets))
    errors = []
    with concurrent.futures.ThreadPoolExecutor(max_workers=max_workers) as executor:
        futures = {executor.submit(download_asset, asset, dest_dir, quiet): asset for asset in assets}
        for future in concurrent.futures.as_completed(futures):
            try:
                future.result()
            except Exception as e:
                if not quiet:
                    print(f"Error downloading asset: {e}")
                errors.append(e)
    if not quiet:
        if not errors:
            print(f"Downloaded {len(assets)} files from '{src}' in repository '{repository}' to '{dest_dir}'")
        else:
            print(f"Downloaded {len(assets) - len(errors)} of {len(assets)} files from '{src}' in repository '{repository}' to '{dest_dir}'. {len(errors)} failed.")
    return len(errors) == 0

def main(args):
    quiet = getattr(args, 'quiet', False)
    success = download_folder(args.src, args.dest, quiet)
    if not success:
        import sys
        sys.exit(1)
