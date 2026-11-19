import assert from 'node:assert/strict';
import { readFileSync } from 'node:fs';
import { resolve } from 'node:path';
import { runInNewContext } from 'node:vm';
import { test } from 'node:test';
import { fileURLToPath } from 'node:url';
const root = resolve(process.argv[2] || fileURLToPath(new URL('./builtin/channel-quality/public', import.meta.url)));
const row = {group:'legacy-enterprise',group_name:'当前企业分组',channel_id:71,channel_name:'验证渠道',channel_type:1,windows:[]};
function loadNative(target, rows=[row], overrides={}) {
 let index=0;
 const effects=[];
 const hooks={Fragment:'fragment',useEffect:fn=>effects.push(fn),useMemo:fn=>fn(),useCallback:fn=>fn,useRef:v=>({current:v}),useState:v=>[Object.hasOwn(overrides,index)?overrides[index++]: (++index,typeof v==='function'?v():v),()=>{}]};
 hooks.default=hooks;
 const jsx=(type,props)=>({type,props:props||{}});
 const t=(s,params={})=>s.replace(/{{(.*?)}}/g,(_,k)=>params[k]??'');
 const component=(name)=>new Proxy(Object.assign(props=>jsx(name,props),{displayName:name}),{get:(fn,key)=>key in fn?fn[key]:component(name+'.'+String(key))});
 const generic=new Proxy({}, {has:()=>true,get:(_,name)=>component(String(name))});
 const requests=[];
 const api={get:async(url,options)=>{requests.push({url,options});return {data:{success:true,data:{items:rows,total:rows.length}}}}};
 const modules=new Proxy({'react':hooks,'react/jsx-runtime':{jsx,jsxs:jsx,Fragment:'fragment'},'react-i18next':{useTranslation:()=>({t,i18n:{language:'zh'}})},'@tanstack/react-query':{useQuery:()=>({data:{items:rows,total:rows.length,groups:rows.map(r=>({code:r.group,name:r.group_name}))},isLoading:false})},'@/lib/api':{api},'../../helpers':{API:api,getAPI:()=>api}}, {get:(o,k)=>o[k]||generic});
 const code=readFileSync(resolve(root,'native',target+'.mjs'),'utf8').replace(/export\s*\{[^}]*\};?\s*$/,'');
 const expose=target==='default'?'({channels:uo,stability:Bo,params:ue,filters:Go,page:er})':'({channels:Ma,stability:Ra,normalize:wt,page:Ha,identity:Na})';
 const context={__NEW_API_EXTENSION_NATIVE_SDK__:{sdk:'v1',platform:target,modules},URLSearchParams,AbortController,Intl,console,performance,window:{},document:{documentElement:{lang:'zh'}}};
 const exports=runInNewContext(code+'\n;'+expose,context);
 return {exports,requests,effects};
}
function collect(node, list=[]) {
 if(node===null||node===undefined||node===false)return list;
 if(Array.isArray(node)){node.forEach(n=>collect(n,list));return list;}
 if(typeof node!=='object'){list.push(String(node));return list;}
 if(typeof node.type==='function'){return collect(node.type(node.props),list);}
 if(node.props){
  if(node.type==='Table'&&node.props.dataSource)for(const row of node.props.dataSource)for(const col of node.props.columns||[])collect(col.render?.(undefined,row)||row[col.dataIndex],list);
  for(const option of node.props.optionList||[])list.push(String(option.label));
  collect(node.props.children,list);
 }
 return list;
}
function nativeView(target,view, rows=[row]){
 const overrides=target==='classic'?(view==='channels'?{3:rows,4:rows.length,5:false}:{4:rows,5:rows.length,6:false}):{};
 const ctx=loadNative(target,rows,overrides);
 return {ctx,tree:ctx.exports[view]({filters:target==='default'?ctx.exports.filters():{},refreshKey:0,makeParams:()=>new URLSearchParams(),queryKey:'fixture',onStatus:()=>{}})};
}
for(const target of ['default','classic']){
 test(target+' 稳定性仅显示当前名称，改名及未知回退',()=>{
  for(const [name,want] of [['当前企业分组','当前企业分组'],['改名企业分组','改名企业分组'],['','legacy-enterprise'],['   ','legacy-enterprise']]){
   const {tree}=nativeView(target,'stability',[{...row,group_name:name}]);
   const text=collect(tree).join(' ');
   assert.ok(text.includes(want),text);
   if(name.trim())assert.ok(!text.includes(row.group),text);
  }
 });
 if(target==='classic')test('Classic 渠道完整多组名称与原始分组分离',()=>{
  const raw={...row,group:'legacy-enterprise,unknown',group_name:'当前企业分组, unknown'};
  const normalized=loadNative('classic').exports.normalize(raw);
  assert.equal(normalized.group,raw.group);
  const {tree}=nativeView('classic','channels',[normalized]);
  const text=collect(tree).join(' ');
  assert.ok(text.includes(raw.group_name),text);
  assert.ok(!text.includes('legacy-enterprise'),text);
 });
}
test('Default 查询继续提交历史code',()=>{
 const {exports:e}=loadNative('default');
 assert.equal(e.params({...e.filters(),group:row.group}).get('groups'),row.group);
});
test('Classic 下拉仅显示名称且value保留code，包括原型名称code',()=>{
 const rows=[row,{...row,group:'__proto__',group_name:'安全名称'}];
 const {exports:e}=loadNative('classic',rows,{6:{channels:[],channelTypes:[],groups:rows.map(r=>({code:r.group,name:r.group_name})),requestedModels:[],upstreamModels:[],outcomes:[]}});
 const tree=e.page();
 const options=[];
 function walk(n){if(!n||typeof n!=='object')return;if(Array.isArray(n)){n.forEach(walk);return;} if(typeof n.type==='function'){walk(n.type(n.props));return;}if(n.props?.optionList)options.push(...n.props.optionList);walk(n.props?.children);}
 walk(tree);
 for(const item of rows){const option=options.find(x=>x.value===item.group);assert.ok(option);assert.equal(option.label,item.group_name);}
});
function loadStatic(){
 const elements=new Map();
 const document={readyState:'loading',addEventListener(){},getElementById(id){if(!elements.has(id))elements.set(id,{innerHTML:'',value:'',disabled:false});return elements.get(id);}};
 let source=readFileSync(resolve(root,'app.js'),'utf8');
 source=source.replace(/\}\)\(\)\s*$/,'globalThis.testAPI={stabilityIdentity,normalizeChannel,renderChannelRows,state,stabilityNodeKey,buildQuery};})()');
 const context={document,window:{addEventListener(){}},console,URLSearchParams,Intl,Date,Map,Set};
 runInNewContext(source,context);
 return {api:context.testAPI,elements};
}
test('兼容页名称展示、转义、空值回退与原始code保留',()=>{
 const {api,elements}=loadStatic();
 for(const name of ['当前企业分组','改名企业分组','<script>危险名称</script>','', '   ']){
  const item={...row,group_name:name};
  const html=api.stabilityIdentity(item);
  if(name.trim())assert.ok(!html.includes('legacy-enterprise'),html);
  else assert.ok(html.includes('legacy-enterprise'),html);
  assert.ok(!html.includes('<script>'),html);
 }
 const channel=api.normalizeChannel({...row,group:'legacy-enterprise,unknown',group_name:'当前企业分组, unknown'});
 assert.equal(channel.group,'legacy-enterprise,unknown');
 api.state.channels=[channel];
 api.renderChannelRows();
 assert.ok(elements.get('channelRows').innerHTML.includes('当前企业分组, unknown'));
 assert.ok(!elements.get('channelRows').innerHTML.includes('legacy-enterprise'));
});
