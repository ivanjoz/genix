import{c as s,i as p,E as y,t as E,r as _}from"./web-D2t8vN-4.js";import{R as x}from"./app-De9iPHfG.js";const u="Invariant Violation",{setPrototypeOf:$=function(e,n){return e.__proto__=n,e}}=Object;class d extends Error{framesToPop=1;name=u;constructor(n=u){super(typeof n=="number"?`${u}: ${n} (see https://github.com/apollographql/invariant-packages)`:n),$(this,d.prototype)}}function c(e,n){if(!e)throw new d(n)}const v=/^[A-Za-z]:\//;function b(e=""){return e&&e.replace(/\\/g,"/").replace(v,n=>n.toUpperCase())}const w=/^[/\\]{2}/,R=/^[/\\](?![/\\])|^[/\\]{2}(?!\.)|^[A-Za-z]:[/\\]/,S=/^[A-Za-z]:$/,I=function(e){if(e.length===0)return".";e=b(e);const n=e.match(w),t=f(e),r=e[e.length-1]==="/";return e=T(e,!t),e.length===0?t?"/":r?"./":".":(r&&(e+="/"),S.test(e)&&(e+="/"),n?t?`//${e}`:`//./${e}`:t&&!f(e)?`/${e}`:e)},h=function(...e){if(e.length===0)return".";let n;for(const t of e)t&&t.length>0&&(n===void 0?n=t:n+=`/${t}`);return n===void 0?".":I(n.replace(/\/\/+/g,"/"))};function T(e,n){let t="",r=0,o=-1,l=0,a=null;for(let i=0;i<=e.length;++i){if(i<e.length)a=e[i];else{if(a==="/")break;a="/"}if(a==="/"){if(!(o===i-1||l===1))if(l===2){if(t.length<2||r!==2||t[t.length-1]!=="."||t[t.length-2]!=="."){if(t.length>2){const g=t.lastIndexOf("/");g===-1?(t="",r=0):(t=t.slice(0,g),r=t.length-1-t.lastIndexOf("/")),o=i,l=0;continue}else if(t.length>0){t="",r=0,o=i,l=0;continue}}n&&(t+=t.length>0?"/..":"..",r=2)}else t.length>0?t+=`/${e.slice(o+1,i)}`:t=e.slice(o+1,i),r=i-o-1;o=i,l=0}else a==="."&&l!==-1?++l:l=-1}return t}const f=function(e){return R.test(e)};function P(e){return`virtual:${e}`}function A(e){return e.handler?.endsWith(".html")?f(e.handler)?e.handler:h(e.root,e.handler):`$vinxi/handler/${e.name}`}const C=new Proxy({},{get(e,n){return c(typeof n=="string","Bundler name should be a string"),{name:n,type:"client",handler:P(A({name:n})),baseURL:"/_build",chunks:new Proxy({},{get(t,r){c(typeof r=="string","Chunk expected");let o=h("/_build",r+".mjs");return{import(){return import(o)},output:{path:o}}}}),inputs:new Proxy({},{get(t,r){c(typeof r=="string","Input must be string");let o=window.manifest[r].output;return{async import(){return import(o)},async assets(){return window.manifest[r].assets},output:{path:o}}}})}}});globalThis.MANIFEST=C;const z=e=>null;var L=E("<span style=font-size:1.5em;text-align:center;position:fixed;left:0px;bottom:55%;width:100%;>");const U=e=>{const n="Error | Uncaught Client Exception";return s(y,{fallback:t=>(console.error(t),[(()=>{var r=L();return p(r,n),r})(),s(z,{code:500})]),get children(){return e.children}})};function m(e){return e.children}function B(){return s(m,{get children(){return s(m,{get children(){return s(U,{get children(){return s(x,{})}})}})}})}function O(e,n){_(e,n)}O(()=>s(B,{}),document.getElementById("app"));const j=void 0;export{j as default};