"""Promote only pilot operations with successful real HTTP/DB contract evidence."""
import copy
import json
from pathlib import Path
root=Path(__file__).resolve().parents[1]
proposal=json.loads((root/'contracts/t01/openapi.proposed.json').read_text())
evidence=json.loads((root/'verification/local-pilot/contract-http-result.json').read_text())
assert evidence['result'].startswith('PASS')
expected={op['operationId'] for methods in proposal['paths'].values() for op in methods.values()}
assert expected==set(evidence['operations'])
runtime_path=root/'cmd/rydo-api/swagger/openapi.json'
old=json.loads(runtime_path.read_text())
runtime=copy.deepcopy(proposal)
runtime['info']={'title':'RYDO API — local pilot implementation','version':'0.2.0-local-pilot','description':'Pilot routes implemented and checked with real HTTP/private PostgreSQL. LINE verification transport is doubled only in tests; real LINE/LIFF and HTTPS are not yet verified. Proposed contract remains in contracts/t01. D01 date correction is not implemented.'}
runtime['x-contract-status']='implemented-local'
runtime['x-runtime-implemented']=True
runtime['x-verification-evidence']='verification/local-pilot/contract-http-result.json'
for methods in runtime['paths'].values():
    for op in methods.values():
        op['summary']=op['summary'].removeprefix('PROPOSED: ')
        op['x-runtime-implemented']=True
        op['x-contract-status']='implemented-local'
        for response in op['responses'].values():
            if response.get('description')=='Proposed success; NOT implemented by runtime.':response['description']='Successful local-pilot response.'
for path in ['/health/live','/health/ready']:
    runtime['paths'][path]=old['paths'][path]
    runtime['paths'][path]['get']['security']=[]
runtime['components']['responses']['Healthy']=old['components']['responses']['Healthy']
runtime['servers']=[{'url':'/','description':'Backend root or same-origin frontend proxy. Configure WEB_APP_URL as the browser origin.'}]
runtime_path.write_text(json.dumps(runtime,ensure_ascii=False,indent=2)+'\n')
print('Promoted 18 verified pilot operations; preserved 2 health operations.')
