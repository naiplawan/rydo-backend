"""Validate the proposed T01 artifact; never send a request or mutate a service."""
import hashlib
import json
import re
from decimal import Decimal, ROUND_HALF_UP
from pathlib import Path
import warnings

from jsonschema import Draft202012Validator, FormatChecker
with warnings.catch_warnings():
    warnings.simplefilter('ignore', DeprecationWarning)
    from jsonschema import RefResolver
from openapi_spec_validator import validate

ROOT = Path(__file__).resolve().parent
spec = json.loads((ROOT / 'openapi.proposed.json').read_text())
validate(spec)
resolver = RefResolver.from_schema(spec)
checks = 0

def check(schema, value, valid=True):
    global checks
    validator = Draft202012Validator(schema, resolver=resolver, format_checker=FormatChecker())
    errors = list(validator.iter_errors(value))
    assert (not errors) == valid, [e.message for e in errors] or 'unexpected acceptance'
    checks += 1


def examples(media):
    if 'example' in media:
        yield media['example']
    for item in media.get('examples', {}).values():
        yield item['value']


def resolve(node):
    if '$ref' not in node:
        return node
    assert node['$ref'].startswith('#/'), 'external refs not allowed'
    out = spec
    for part in node['$ref'][2:].split('/'):
        out = out[part.replace('~1', '/').replace('~0', '~')]
    return out


def walk(node):
    if isinstance(node, dict):
        if '$ref' in node:
            resolve(node)
        for key, value in node.items():
            if key not in ('example', 'examples'):
                walk(value)
    elif isinstance(node, list):
        for value in node:
            walk(value)

walk(spec)
operations = []
for path, methods in spec['paths'].items():
    assert path.startswith('/api/v1/')
    for method, op in methods.items():
        assert op['x-runtime-implemented'] is False
        operations.append(op['operationId'])
        if 'requestBody' in op:
            for media in op['requestBody']['content'].values():
                for value in examples(media):
                    check(media['schema'], value)
        for response in op['responses'].values():
            response = resolve(response)
            for media in response.get('content', {}).values():
                for value in examples(media):
                    check(media['schema'], value)
        if method in ('post', 'patch') and op['operationId'] in ('confirmCreate', 'confirmEdit'):
            names = {resolve(p)['name'] for p in op['parameters']}
            assert {'X-Preview-Token', 'Idempotency-Key'} <= names
        if not path.startswith('/api/v1/auth/'):
            assert op['security'] == [{'BearerAuth': []}]
assert len(operations) == len(set(operations)) == 18

example_file = json.loads((ROOT / 'examples.json').read_text())
cases = example_file['cases']
sequence = example_file['confirmation_sequence']
assert sequence[1]['body'] == sequence[2]['body']
assert sequence[1]['headers'] == sequence[2]['headers']
for step in sequence[:3]:
    check({'$ref':'#/components/schemas/SessionCreate'}, step['body'])
    if step['step'] != 'preview':
        for name in ('IdempotencyKey', 'PreviewToken'):
            parameter = spec['components']['parameters'][name]
            check(parameter['schema'], step['headers'][parameter['name']])
for case in cases:
    check({'$ref': '#/components/schemas/' + case['schema']}, case['value'], case['valid'])

# Compare all eight concrete pilot examples in the authoritative API document.
backend = ROOT.parents[1]
api_text = (backend.parent / 'rydo-doc/04_API_SPECIFICATION.md').read_text()
source_examples = [json.loads(b) for b in re.findall(r'```json\n(.*?)\n```', api_text, re.S)]
assert len(source_examples) == 10
mapping = {1:'Error', 2:'Login', 3:'AuthResponse', 4:'ProfileWrite',
           5:'PlatformsResponse', 6:'VehicleCreate', 7:'SessionCreate', 8:'DailyReportResponse'}
def replace_uuid(value):
    if isinstance(value, dict):
        return {k: replace_uuid(v) for k, v in value.items()}
    if isinstance(value, list):
        return [replace_uuid(v) for v in value]
    return '11111111-1111-4111-8111-111111111111' if value == 'uuid' else value
for index, schema in mapping.items():
    check({'$ref':'#/components/schemas/' + schema}, replace_uuid(source_examples[index]))
# Generic envelope (#0) is not a concrete endpoint; preferences (#9) are post-MVP.
assert source_examples[8] == spec['paths']['/api/v1/reports/daily']['get']['responses']['200']['content']['application/json']['example']

# Independently verify documented workday numbers; not a runtime/API test.
r = lambda x: x.quantize(Decimal('.01'), rounding=ROUND_HALF_UP)
D = Decimal
fuel = r(D(100) / D(40) * D(35))
reserve = r(D(100) * D('.600'))
net = D(850) - fuel - reserve - D(30)
assert (fuel, reserve, net, r(net / D(8)), r(net / D(100))) == tuple(map(D, ['87.50', '60.00', '672.50', '84.06', '6.73']))
assert (r(D('6.725')), r(D('-6.725'))) == (D('6.73'), D('-6.73'))
assert (r(D('702.50') / D('7.75')), r(D('702.50') / D(100))) == (D('90.65'), D('7.03'))

# Source-only separation proof, including runtime schema hash captured before writing.
backend = ROOT.parents[1]
baseline = json.loads((ROOT / 'source-reviewed.json').read_text())
unchanged = []
for relative, expected in baseline.items():
    if relative.endswith('Task.md'):
        continue  # T01 result may be linked in its own task entry.
    current = hashlib.sha256((backend.parent / relative).read_bytes()).hexdigest()
    assert current == expected, ('source changed: rebase review before accepting', relative)
    unchanged.append(relative)
runtime = json.loads((backend / 'cmd/rydo-api/swagger/openapi.json').read_text())
assert set(runtime['paths']) == {'/health/live', '/health/ready'}
print(json.dumps({'openapi':'PASS 3.1.0 structure + local refs', 'operations':len(operations), 'schema_example_checks':checks, 'case_count':len(cases), 'source_json_parsed':len(source_examples), 'source_pilot_examples_validated':len(mapping), 'independent_decimal_checks':'PASS', 'unchanged_source_files':len(unchanged), 'runtime_schema_paths':list(runtime['paths']), 'runtime_api_tests':'NOT RUN; proposed contract only'}, indent=2))
