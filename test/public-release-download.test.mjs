import assert from 'node:assert/strict';
import {test} from 'node:test';
import {mkdtemp, readFile, readdir, rm} from 'node:fs/promises';
import {createHash} from 'node:crypto';
import os from 'node:os';
import path from 'node:path';
import {downloadPublicRelease} from '../scripts/download-public-release.mjs';

const prefix = 'https://api.github.com/repos/puretokensx/puretokens-skill/releases';
const names = ['release-manifest.json', 'puretokens-skill-fetch.sh', 'puretokens-skill-fetch.ps1',
  'puretokens-skill-install.sh', 'puretokens-skill-install.ps1',
  ...['darwin-amd64','darwin-arm64','linux-amd64','linux-arm64','windows-amd64','windows-arm64'].map(p=>'puretokens-skill-1.2.3-'+p+'.zip')];
function fixture(mode) {
  const bytes = Buffer.from('verified fixture');
  const release = {tag_name:'v1.2.3', draft:false, prerelease:false, assets:names.map((name,i)=>({
    id:100+i,name,state:'uploaded',size:bytes.length,digest:'sha256:'+createHash('sha256').update(bytes).digest('hex'),
    url:prefix+'/assets/'+(100+i),browser_download_url:'https://github.com/puretokensx/puretokens-skill/releases/download/v1.2.3/'+name
  }))};
  if(mode==='wrong-tag') release.tag_name='v9.9.9';
  if(mode==='draft') release.draft=true;
  if(mode==='duplicate') release.assets[1]=release.assets[0];
  if(mode==='wrong-origin') release.assets[0].url='https://example.invalid/asset';
  if(mode==='bad-digest') release.assets[0].digest='sha256:'+'0'.repeat(64);
  if(mode==='missing') release.assets.pop();
  const calls=[];
  return {calls, request:async (url,accept,limit)=>{
    calls.push({url,accept,limit});
    if(url===prefix+'/latest') return Buffer.from(JSON.stringify(release));
    assert.ok(url.startsWith(prefix+'/assets/'));
    assert.equal(accept,'application/octet-stream');
    if(mode==='download-failure') throw new Error('HTTP 504');
    return bytes;
  }};
}
test('anonymous release verifier downloads exact stable assets without auth or browser routes',async t=>{
  const root=await mkdtemp(path.join(os.tmpdir(),'pt-public-download-'));
  t.after(()=>rm(root,{recursive:true,force:true}));
  const f=fixture('valid');
  await downloadPublicRelease('v1.2.3',path.join(root,'assets'),{request:f.request});
  assert.equal(f.calls.length,12);
  assert.deepEqual((await readdir(path.join(root,'assets'))).sort(),[...names].sort());
  assert.equal(await readFile(path.join(root,'assets','puretokens-skill-fetch.sh'),'utf8'),'verified fixture');
});
for(const mode of ['wrong-tag','draft','duplicate','wrong-origin','bad-digest','missing','download-failure']) {
  test('anonymous release verifier stops on '+mode+' without retry or partial output',async t=>{
    const root=await mkdtemp(path.join(os.tmpdir(),'pt-public-download-'));
    t.after(()=>rm(root,{recursive:true,force:true}));
    const f=fixture(mode);
    await assert.rejects(downloadPublicRelease('v1.2.3',path.join(root,'assets'),{request:f.request}));
    assert.ok(f.calls.length<=2);
    assert.deepEqual(await readdir(root),[]);
  });
}
