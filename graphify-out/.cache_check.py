import json
from graphify.cache import check_semantic_cache
from pathlib import Path

SPEC_PATH = r"C:\Users\123\.claude\skills\graphify\references\extraction-spec.md"

detect = json.loads(Path('graphify-out/.graphify_detect.json').read_text(encoding='utf-8-sig'))
all_files = [f for cat in ('document', 'paper', 'image') for f in detect['files'].get(cat, [])]

cached_nodes, cached_edges, cached_hyperedges, uncached = check_semantic_cache(all_files, root='.', prompt_file=SPEC_PATH)

if cached_nodes or cached_edges or cached_hyperedges:
    Path('graphify-out/.graphify_cached.json').write_text(json.dumps({'nodes': cached_nodes, 'edges': cached_edges, 'hyperedges': cached_hyperedges}, ensure_ascii=False), encoding='utf-8')
else:
    Path('graphify-out/.graphify_cached.json').unlink(missing_ok=True)
Path('graphify-out/.graphify_uncached.txt').write_text('\n'.join(uncached), encoding='utf-8')
print(f'Cache: {len(all_files)-len(uncached)} files hit, {len(uncached)} files need extraction')
