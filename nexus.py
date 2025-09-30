import os
import requests
from typing import Optional, List

# Configuration - update these as needed
NEXUS_URL = os.environ.get("NEXUS_URL", "http://localhost:8081")
REPOSITORY = os.environ.get("NEXUS_REPO", "builds")
USERNAME = os.environ.get("NEXUS_USER", "admin")
PASSWORD = os.environ.get("NEXUS_PASS", "admin")

UPLOAD_ENDPOINT = f"{NEXUS_URL}/service/rest/v1/components?repository={REPOSITORY}"

def collect_files(directory: str) -> List[str]:
    """
    Recursively collect all file paths from the directory.
    """
    file_paths = []
    for root, _, files in os.walk(directory):
        for file in files:
            file_paths.append(os.path.join(root, file))
    return file_paths

def upload_files(directory: str, subdir: Optional[str] = None):
    """
    Upload all files from a directory (recursively) to Nexus RAW repository in a single HTTP call.
    """
    file_paths = collect_files(directory)
    files = {}
    for idx, file_path in enumerate(file_paths, 1):
        rel_path = os.path.relpath(file_path, directory)
        rel_path = rel_path.replace(os.sep, "/")  # Ensure forward slashes
        files[f'raw.asset{idx}'] = (os.path.basename(file_path), open(file_path, 'rb'))
        files[f'raw.asset{idx}.filename'] = (None, rel_path)
    files['raw.directory'] = (None, subdir)
    response = requests.post(
        UPLOAD_ENDPOINT,
        auth=(USERNAME, PASSWORD),
        files=files
    )
    # Close all opened files
    for idx in range(1, len(file_paths) + 1):
        file_obj = files[f'raw.asset{idx}'][1]
        file_obj.close()
    if response.status_code == 204:
        print(f"Uploaded {len(file_paths)} files from {directory}")
    else:
        print(f"Failed to upload files: {response.status_code} {response.text}")

def main():
    import argparse
    parser = argparse.ArgumentParser(description="Upload all files from a directory to Nexus RAW repository in a single HTTP call.")
    parser.add_argument("directory", help="Directory to upload")
    parser.add_argument("--subdir", help="Optional subdirectory in Nexus RAW repo", default=None)
    args = parser.parse_args()
    upload_files(args.directory, args.subdir)

if __name__ == "__main__":
    main()
