import os
import sys
import requests
from typing import Optional, List
from tqdm import tqdm
from requests_toolbelt.multipart.encoder import MultipartEncoder, MultipartEncoderMonitor

NEXUS_URL = os.environ.get("NEXUS_URL", "http://localhost:8081")
USERNAME = os.environ.get("NEXUS_USER", "admin")
PASSWORD = os.environ.get("NEXUS_PASS", "admin")

def collect_files(src: str) -> List[str]:
    file_paths = []
    for root, _, files in os.walk(src):
        for file in files:
            file_paths.append(os.path.join(root, file))
    return file_paths

def upload_files(src: str, repository: str, subdir: Optional[str] = None, quiet: bool = False):
    file_paths = collect_files(src)
    fields = {}
    for idx, file_path in enumerate(file_paths, 1):
        rel_path = os.path.relpath(file_path, src)
        rel_path = rel_path.replace(os.sep, "/")
        fields[f'raw.asset{idx}'] = (os.path.basename(file_path), open(file_path, 'rb'))
        fields[f'raw.asset{idx}.filename'] = (None, rel_path)
    fields['raw.directory'] = (None, subdir)

    upload_endpoint = f"{NEXUS_URL}/service/rest/v1/components?repository={repository}"
    encoder = MultipartEncoder(fields=fields)
    
    # Show progress bar only if stdout is a TTY and not in quiet mode
    show_progress = sys.stdout.isatty() and not quiet
    progress = tqdm(total=encoder.len, unit='B', unit_scale=True, desc='Uploading', disable=not show_progress)

    def callback(monitor):
        progress.update(monitor.bytes_read - progress.n)

    monitor = MultipartEncoderMonitor(encoder, callback)
    headers = {'Content-Type': monitor.content_type}
    response = requests.post(
        upload_endpoint,
        auth=(USERNAME, PASSWORD),
        data=monitor,
        headers=headers
    )
    progress.close()
    for key, value in fields.items():
        if isinstance(value, tuple) and hasattr(value[1], 'close'):
            value[1].close()
    if not quiet:
        if response.status_code == 204:
            print(f"Uploaded {len(file_paths)} files from {src}")
        else:
            print(f"Failed to upload files: {response.status_code} {response.text}")

def main(args):
    if "/" in args.dest:
        repository, subdir = args.dest.split("/", 1)
    else:
        repository, subdir = args.dest, None
    quiet = getattr(args, 'quiet', False)
    upload_files(args.src, repository, subdir, quiet)
