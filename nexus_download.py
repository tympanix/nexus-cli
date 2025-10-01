import os
import requests
from typing import Optional
from tqdm import tqdm
import concurrent.futures

NEXUS_URL = os.environ.get("NEXUS_URL", "http://localhost:8081")
USERNAME = os.environ.get("NEXUS_USER", "admin")
PASSWORD = os.environ.get("NEXUS_PASS", "admin")

SEARCH_ENDPOINT = f"{NEXUS_URL}/service/rest/v1/search/assets"

def list_assets(repository: str, folder: str) -> list:
    continuation_token = None
    assets = []
    params = {
        'repository': repository,
        'format': 'raw',
        'direction': 'asc',
        'sort': 'name',
        'q': f"/{folder}/*"
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

def download_asset(asset: dict, dest_dir: str):
    download_url = asset['downloadUrl']
    path = asset['path'].lstrip("/")
    local_path = os.path.join(dest_dir, path)
    os.makedirs(os.path.dirname(local_path), exist_ok=True)
    with requests.get(download_url, auth=(USERNAME, PASSWORD), stream=True) as r:
        r.raise_for_status()
        total = int(r.headers.get('content-length', 0))
        with open(local_path, 'wb') as f, tqdm(
            desc=f"Downloading {path}",
            total=total,
            unit='B',
            unit_scale=True
        ) as bar:
            for chunk in r.iter_content(chunk_size=8192):
                if chunk:
                    f.write(chunk)
                    bar.update(len(chunk))

def download_folder(folder_arg: str, dest_dir: str) -> bool:
    if '/' not in folder_arg:
        print("Error: The folder argument must be in the form 'repository/folder' or 'repository/folder/subfolder'.")
        return False
    repository, folder = folder_arg.split('/', 1)
    assets = list_assets(repository, folder)
    if not assets:
        print(f"No assets found in folder '{folder}' in repository '{repository}'")
        return True
    max_workers = min(8, len(assets))
    errors = []
    with concurrent.futures.ThreadPoolExecutor(max_workers=max_workers) as executor:
        futures = {executor.submit(download_asset, asset, dest_dir): asset for asset in assets}
        for future in concurrent.futures.as_completed(futures):
            try:
                future.result()
            except Exception as e:
                print(f"Error downloading asset: {e}")
                errors.append(e)
    if not errors:
        print(f"Downloaded {len(assets)} files from '{folder}' in repository '{repository}' to '{dest_dir}'")
    else:
        print(f"Downloaded {len(assets) - len(errors)} of {len(assets)} files from '{folder}' in repository '{repository}' to '{dest_dir}'. {len(errors)} failed.")
    return len(errors) == 0

def main(args):
    success = download_folder(args.folder, args.dest)
    if not success:
        import sys
        sys.exit(1)
