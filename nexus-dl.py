import os
import requests
from typing import Optional
from tqdm import tqdm
import concurrent.futures

# Configuration - update these as needed
NEXUS_URL = os.environ.get("NEXUS_URL", "http://localhost:8081")
REPOSITORY = os.environ.get("NEXUS_REPO", "builds")
USERNAME = os.environ.get("NEXUS_USER", "admin")
PASSWORD = os.environ.get("NEXUS_PASS", "admin")

SEARCH_ENDPOINT = f"{NEXUS_URL}/service/rest/v1/search/assets"

def list_assets(folder: str) -> list:
    """
    List all assets in a given folder (recursively) in the Nexus RAW repository.
    """
    continuation_token = None
    assets = []
    params = {
        'repository': REPOSITORY,
        'format': 'raw',
        'direction': 'asc',
        'sort': 'name',
        'q': f"{folder}/*"  # Use glob pattern with 'q' param
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
    """
    Download a single asset to the destination directory, preserving the folder structure.
    """
    download_url = asset['downloadUrl']
    path = asset['path'].lstrip("/")  # Normalize to relative path
    local_path = os.path.join(dest_dir, path)  # Preserve subfolders
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

def download_folder(folder: str, dest_dir: str):
    """
    Download all assets in a Nexus RAW folder recursively to dest_dir.
    """
    assets = list_assets(folder)
    if not assets:
        print(f"No assets found in folder '{folder}'")
        return
    max_workers = min(8, len(assets))  # Limit number of threads
    with concurrent.futures.ThreadPoolExecutor(max_workers=max_workers) as executor:
        futures = [executor.submit(download_asset, asset, dest_dir) for asset in assets]
        for future in concurrent.futures.as_completed(futures):
            try:
                future.result()
            except Exception as e:
                print(f"Error downloading asset: {e}")
    print(f"Downloaded {len(assets)} files from '{folder}' to '{dest_dir}'")

def main():
    import argparse
    parser = argparse.ArgumentParser(description="Download all files from a Nexus RAW folder recursively.")
    parser.add_argument("folder", help="Nexus RAW folder to download (e.g. 'myfolder' or 'myfolder/subfolder')")
    parser.add_argument("dest", help="Destination directory to save files")
    args = parser.parse_args()
    download_folder(args.folder, args.dest)

if __name__ == "__main__":
    main()
