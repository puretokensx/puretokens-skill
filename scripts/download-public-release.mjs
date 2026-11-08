// Maintainer-only public-download gate. Never distributed as a user runtime.
import {execFile} from 'node:child_process';
import {promisify} from 'node:util';
import {createHash} from 'node:crypto';
import {mkdir, rm, writeFile} from 'node:fs/promises';
import path from 'node:path';
import {fileURLToPath} from 'node:url';

const run = promisify(execFile);
const api = 'https://api.github.com/repos/puretokensx/puretokens-skill/releases';
async function anonymousRequest(url, accept, limit) {
  try {
    const result = await run('curl', ['-q','--fail','--silent','--show-error','--location',
      '--proto','=https','--proto-redir','=https','--max-redirs','5',
      '--connect-timeout','15','--max-time','60','--max-filesize',String(limit),
      '--header','Accept: '+accept,'--user-agent','puretokens-release-verifier',url],
    {encoding:'buffer',maxBuffer:limit,timeout:65000});
    return result.stdout;
  } catch (error) {
    const code = Number.isInteger(error.code) ? error.code : 'unconfirmed';
    throw new Error('Anonymous official asset download failed (curl_exit=' + code + '); no retry or publication mutation.');
  }
}

export async function downloadPublicRelease(tag, directory, {request=anonymousRequest}={}) {
  if (!/^v[0-9]+[.][0-9]+[.][0-9]+$/.test(tag)) throw new Error('Expected a pinned stable tag.');
  const metadata=JSON.parse((await request(api+'/latest','application/vnd.github+json',2097152)).toString('utf8'));
  const names=['release-manifest.json','puretokens-skill-fetch.sh','puretokens-skill-fetch.ps1',
    'puretokens-skill-install.sh','puretokens-skill-install.ps1',
    ...['darwin-amd64','darwin-arm64','linux-amd64','linux-arm64','windows-amd64','windows-arm64'].map(p=>'puretokens-skill-'+tag.slice(1)+'-'+p+'.zip')];
  if(metadata.tag_name!==tag || metadata.draft!==false || metadata.prerelease!==false ||
      !Array.isArray(metadata.assets) || metadata.assets.length!==names.length) throw new Error('Latest stable release does not match the expected complete release.');
  const assets=names.map(name=>{
    const matches=metadata.assets.filter(a=>a.name===name);
    if(matches.length!==1) throw new Error('Release asset is missing or ambiguous.');
    const a=matches[0];
    if(!Number.isSafeInteger(a.id) || a.id<=0 || a.state!=='uploaded' ||
        !Number.isSafeInteger(a.size) || a.size<=0 || a.size>33554432 || !/^sha256:[0-9a-f]{64}$/.test(a.digest) ||
        a.url!==api+'/assets/'+a.id || a.browser_download_url!=='https://github.com/puretokensx/puretokens-skill/releases/download/'+tag+'/'+name) throw new Error('Release asset identity is invalid.');
    return a;
  });
  await mkdir(directory,{mode:0o700});
  try {
    for(const asset of assets) {
      const bytes=await request(api+'/assets/'+asset.id,'application/octet-stream',asset.size);
      if(bytes.length!==asset.size || 'sha256:'+createHash('sha256').update(bytes).digest('hex')!==asset.digest) throw new Error('Anonymous asset size or digest mismatch.');
      await writeFile(path.join(directory,asset.name),bytes,{flag:'wx',mode:0o600});
    }
  } catch(error) { await rm(directory,{recursive:true,force:true}); throw error; }
  return {tag,assets:assets.length};
}

if(process.argv[1] && path.resolve(process.argv[1])===fileURLToPath(import.meta.url)) {
  try {
    if(process.argv.length!==4) throw new Error('Usage: node scripts/download-public-release.mjs <tag> <new-output-directory>');
    const result=await downloadPublicRelease(process.argv[2],path.resolve(process.argv[3]));
    console.log('Verified '+result.assets+' anonymous official asset downloads for '+result.tag+'.');
  } catch(error) { console.error(error.message); process.exitCode=1; }
}
