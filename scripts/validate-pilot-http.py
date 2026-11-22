"""Compare captured real HTTP/DB test traffic to the accepted T01 contract."""
import json
import re
import warnings
from pathlib import Path
from jsonschema import Draft202012Validator, FormatChecker
with warnings.catch_warnings():
    warnings.simplefilter('ignore', DeprecationWarning)
    from jsonschema import RefResolver
from openapi_spec_validator import validate
root=Path(__file__).resolve().parents[1]
spec=json.loads((root/'contracts/t01/openapi.proposed.json').read_text())
validate(spec)
resolver=RefResolver.from_schema(spec)
rows=json.loads((root/'verification/local-pilot/http-capture.json').read_text())
covered=set();checks=0

def node(v):
    if '$ref' not in v:return v
    value=spec
    for part in v['$ref'][2:].split('/'):value=value[part]
    return value

def check(schema,value,context):
    global checks
    errors=list(Draft202012Validator(schema,resolver=resolver,format_checker=FormatChecker()).iter_errors(value))
    assert not errors,(context,[e.message for e in errors])
    checks+=1

for row in rows:
    path=row['path'].split('?')[0];matched=None
    for template,methods in spec['paths'].items():
        pattern=re.sub(r'\{[^}]+\}',r'[^/]+',template)
        if re.fullmatch(pattern,path):matched=(template,methods);break
    assert matched,('uncontracted HTTP path',path)
    template,methods=matched;op=methods[row['method'].lower()];status=str(row['status'])
    assert status in op['responses'],(path,status)
    response=node(op['responses'][status]);media=response.get('content',{}).get('application/json')
    if media:check(media['schema'],row['response'],(row['method'],path,status))
    else:assert row['response']=={},('unexpected response body',path,status)
    assert row['cache_control']=='no-store'
    if response.get('headers',{}).get('Set-Cookie',{}).get('required'):assert row['set_cookie_present'],('missing Set-Cookie',path)
    if row['status']<300:
        covered.add(op['operationId'])
        if 'requestBody' in op:check(op['requestBody']['content']['application/json']['schema'],row['request'],(path,'request'))
expected={op['operationId'] for methods in spec['paths'].values() for op in methods.values()}
assert covered==expected,('untested successful operations',sorted(expected-covered))
print(json.dumps({'contract':'T01 accepted P01-P08','wire_checks':checks,'http_responses':len(rows),'successful_operations':len(covered),'operations':sorted(covered),'result':'PASS; real HTTP + private PostgreSQL; LINE transport is test-only double'},indent=2))
