"""Validate actual OCI server responses against the Smithy-derived OpenAPI."""
import concurrent.futures
import json
import time
import urllib.request
import urllib.error
from pathlib import Path
from jsonschema import Draft202012Validator, FormatChecker

spec = json.loads(Path('/workspace/api/generated/openapi/sintonizados.openapi.json').read_text())
base = 'http://127.0.0.1:8080'
seen = set()

def check(method, path, model_path=None, body=None, headers=None, expected=200):
    req = urllib.request.Request(base + path, data=body, method=method, headers=headers or {})
    try:
        response = urllib.request.urlopen(req, timeout=10)
    except urllib.error.HTTPError as error:
        response = error
    with response:
        assert response.status == expected, (path, response.status, response.read())
        operation = spec['paths'][model_path or path][method.lower()]
        content = operation['responses'][str(expected)].get('content', {})
        mime = response.headers['Content-Type'].split(';')[0]
        assert mime in content, (path, mime, content)
        if mime == 'text/event-stream':
            text = response.read(1)
            assert text
        else:
            raw = response.read()
            value = json.loads(raw) if mime == 'application/json' else raw.decode()
            schema = dict(content[mime]['schema'])
            schema['components'] = spec.get('components', {})
            Draft202012Validator(schema, format_checker=FormatChecker()).validate(value)
            seen.add(operation['operationId'])
            return value
    seen.add(operation['operationId'])

for attempt in range(50):
    try:
        check('GET', '/health')
        break
    except OSError:
        time.sleep(.1)
else:
    raise RuntimeError('server did not start')

auth = {'Authorization': 'Bearer contract-test', 'Content-Type': 'application/json'}
check('GET', '/')
check('GET', '/operator')
check('GET', '/assets/operator.js', '/assets/{name}')
check('GET', '/assets/mic-worklet.js', '/assets/{name}')
check('POST', '/api/sessions', body=b'{}', expected=401, headers={'Content-Type': 'application/json'})
for sid in ('contract-a', 'contract-b'):
    check('POST', '/api/sessions', body=json.dumps({'session_id': sid, 'title': sid, 'language': 'en'}).encode(), headers=auth, expected=201)

def feed(sid):
    for n in range(1, 11):
        check('POST', f'/api/sessions/{sid}/audio', '/api/sessions/{session_id}/audio', body=b'\0' * 3200,
              headers={**auth, 'Content-Type': 'application/octet-stream', 'X-Audio-Sequence': str(n)}, expected=202)
        time.sleep(.03)
    check('POST', f'/api/sessions/{sid}/end', '/api/sessions/{session_id}/end', headers=auth)

with concurrent.futures.ThreadPoolExecutor(max_workers=2) as pool:
    list(pool.map(feed, ('contract-a', 'contract-b')))
for sid in ('contract-a', 'contract-b'):
    detail = check('GET', f'/api/sessions/{sid}', '/api/sessions/{session_id}')
    assert detail['status'] == 'ended' and detail['subtitle_count'] == 10
    check('GET', f'/talks/{sid}', '/talks/{session_id}')
    check('GET', f'/obs/{sid}', '/obs/{session_id}')
    result = check('GET', f'/api/sessions/{sid}/subtitles', '/api/sessions/{session_id}/subtitles')
    assert len(result['subtitles']) == 10
    assert all(sub['session_id'] == sid for sub in result['subtitles'])
    check('GET', f'/api/sessions/{sid}/events', '/api/sessions/{session_id}/events', headers={'Last-Event-ID': '9'})
check('GET', '/api/sessions')
check('GET', '/metrics', headers=auth)
check('GET', '/api/sessions/missing', '/api/sessions/{session_id}', expected=404)
expected = {op['operationId'] for methods in spec['paths'].values() for op in methods.values()}
assert seen == expected, expected - seen
print(f'PASS: {len(seen)} modeled operations; real OCI responses match OpenAPI 3.1; two concurrent sessions.')
