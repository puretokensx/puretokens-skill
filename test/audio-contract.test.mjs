import assert from "node:assert/strict";
import {readFile, mkdir, mkdtemp, cp, writeFile, rm} from "node:fs/promises";
import path from "node:path";
import os from "node:os";
import test from "node:test";
import {repositoryRoot} from "../scripts/skill-registry.mjs";
import {syncSkillGuidance} from "../scripts/sync-skill-guidance.mjs";
import {compileSchema} from "./support/schema-validator.mjs";
const json = async file => JSON.parse(await readFile(path.join(repositoryRoot,file),"utf8"));

test("reviewed embedded audio profiles, installed selection and request schema agree", async () => {
 const source=await json("runtime/executor/audio-profiles.json");
 const index=await json("skills/puretokens-audio/references/model-index.json");
 const schema=await json("schemas/audio-request.schema.json");
 const validate=compileSchema(schema,[schema]);
 const musicSchema=await json("schemas/executor-request.schema.json");
 const musicValidate=compileSchema(musicSchema,[musicSchema]);
 assert.equal(index.reviewedAt,source.reviewedAt);
 assert.deepEqual(index.models.map(m=>m.id),Object.keys(source.models));
 assert.equal(index.scope,"reviewed_contract_not_live_availability");
 for(const item of index.models){
  const profile=source.models[item.id];
  const installed=await json(`skills/puretokens-audio/references/${item.profile}`);
  assert.deepEqual(installed,{schemaVersion:1,reviewedAt:source.reviewedAt,...profile});
  if(profile.operation==="music"){
   assert.equal(profile.executor_kind,"music");
   assert.equal(profile.max_output_bytes,32*1024*1024);
   const rules=musicSchema.allOf.find(rule=>rule.if?.properties?.kind?.const==="music").then;
   const submission=rules.allOf[0].else.properties;
   assert.equal(rules.properties.model.const,item.id);
   assert.equal(submission.prompt.maxLength,profile.max_prompt_characters);
   assert.equal(submission.parameters.properties.lyrics.maxLength,profile.max_lyrics_characters);
   assert.deepEqual(submission.parameters.properties.response_format.enum,profile.formats);
   assert.deepEqual(musicValidate({kind:"music",operation:"generate",model:item.id,prompt:"Music",parameters:{instrumental:true,response_format:"mp3"},output_dir:"/output"}),[]);
   assert.ok(validate({operation:"music",model:item.id}).length,"music must not enter synchronous audio");
   continue;
  }
  const variant=schema.oneOf.find(v=>v.properties.model.const===item.id);
  assert.equal(variant.properties.operation.const,profile.operation);
  let request={operation:profile.operation,model:item.id};
  if(profile.operation==="speech"){
   if (profile.voice_pattern) assert.equal(variant.properties.voice.pattern, profile.voice_pattern+"(?![\\s\\S])");
   else assert.deepEqual(variant.properties.voice.enum,Object.keys(profile.voices));
   assert.equal(variant.properties.input.maxLength,profile.max_input_characters);
   assert.equal(variant.properties.speed?.minimum,profile.speed_min);
   assert.equal(variant.properties.speed?.maximum,profile.speed_max);
   assert.equal(!!variant.properties.instruction,profile.supports_instruction);
   request={...request,input:"text",voice:profile.voice_pattern?"JBFqnCBsd6RMkjVDRZzb":"cixingnansheng",response_format:profile.formats[0],output_dir:"/output"};
  }else if(profile.operation==="transcribe"){
   assert.equal(profile.max_input_bytes,8*1024*1024);request.file="/recording.wav";
  }else if(profile.operation==="voice_change"){
   request={...request,file:"/recording.wav",voice:"JBFqnCBsd6RMkjVDRZzb",response_format:"mp3",output_dir:"/output"};
  }else{
   assert.equal(variant.properties.instruction.maxLength,profile.max_instruction_characters);
   request={...request,instruction:"rain",response_format:"mp3",output_dir:"/output"};
   if(profile.duration_max) request.duration_seconds=profile.duration_min;
   if(profile.instrumental_required) request.instrumental=false;
  }
  assert.deepEqual(validate(request),[],item.id);
 }
});

test("audio generated profile drift cannot pass synchronization checks", async t=>{
 const root=await mkdtemp(path.join(os.tmpdir(),"pt-audio-guidance-"));
 t.after(()=>rm(root,{recursive:true,force:true}));
 for(const dir of ["skills","references"]){await cp(path.join(repositoryRoot,dir),path.join(root,dir),{recursive:true});}
 await mkdir(path.join(root,"runtime/executor"),{recursive:true});
 await cp(path.join(repositoryRoot,"runtime/executor/audio-profiles.json"),path.join(root,"runtime/executor/audio-profiles.json"),{recursive:true});
 const file="skills/puretokens-audio/references/profiles/stepaudio-2.5-tts.json";
 const original=await readFile(path.join(root,file),"utf8");
 await writeFile(path.join(root,file),original.replace('"max_input_characters": 1000','"max_input_characters": 9999'));
 assert.ok((await syncSkillGuidance(root)).includes(file));
 await syncSkillGuidance(root,{write:true});
 assert.equal(await readFile(path.join(root,file),"utf8"),original);
});
