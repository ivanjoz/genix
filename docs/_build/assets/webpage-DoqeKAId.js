import{a as w,g as V,c as m,d as D,f as g,i as p,F as z,b as h,s as F,t as S,e as v}from"./web-B5q02dNy.js";import{u as W,c as j,i as L,p as O,j as q,P as G,k as R,l as A,I as H,s as u}from"./app-BWK4gxn2.js";var J=S('<div class="h100 p-rel cms-1">'),k=S("<div>"),K=S("<div>Seleccione una sección para editar su contenido."),Q=S("<div>--"),U=S('<div><div><i class="icon-plus h6"></i>Sección</div><div>'),X=S("<i class=icon-up-1>"),Y=S("<i class=icon-down-1>");const[ne,ae]=w({}),M=new Map,Z=r=>{if(M.size===0)for(let s of R)M.set(s.type,s);return M.get(r)};function oe(){const r=W(),[s,o]=w(O),[i,f]=w(),[a,b]=w([]),P=j(L,"id");V(()=>{console.log(r.pathname)});const y=(t,e)=>{const n=[],l={type:1,content:"demo",title:"demo"},c=s();console.log("args sections::",t,e);debugger;for(let d=0;d<c.length;d++)e===1&&t===d&&n.push(l),n.push(c[d]),e===2&&t===d&&n.push(l);console.log("nuevas seccciones::",n),o(n)},x=(t,e)=>{let n=!1;if((t===s().length-1&&e===2||t===0&&e===1)&&(n=!0),n)return;const l=s()[t],c=s().filter(d=>d!==l);e===1?t--:e===2&&t++,c.splice(t,0,l),o(c)},N=t=>{const e=[];for(const n of s())n.id===t.id?e.push(t):e.push(n);f(t),o(e)};return m(G,{title:"Webpage",class:"",views:[[1,"Editor"],[2,"Secciones"]],pageStyle:{display:"grid","grid-template-columns":"1.1fr 4fr",padding:"0","background-color":"white"},get children(){return[g(()=>g(()=>A()===1)()&&(()=>{var t=k();return p(t,(()=>{var e=g(()=>!i());return()=>e()&&K()})(),null),p(t,m(z,{get each(){return a()},children:e=>e.type===1?m(H,{css:"mb-10",inputCss:"s5",get label(){return e.name},save:"content",saveOn:e,onChange:()=>{console.log(e.content);const n={...i()};n[e.key]=e.content,N(n)}}):Q()}),null),h(()=>v(t,`h100 px-08 py-08 p-rel flex-column ${u.webpage_dev_card}`)),t})()),g(()=>g(()=>A()===2)()&&(()=>{var t=k();return p(t,()=>R.map(e=>{const n=g(()=>i()?.type===e.type);return m(I,{params:e,get isSelected(){return n()},onSelect:()=>{if(i()){const l={...i()};l.type=e.type,N(l)}}})})),h(()=>v(t,`h100 px-08 py-08 p-rel flex-column ${u.webpage_dev_card}`)),t})()),(()=>{var t=J();return t.style.setProperty("overflow","auto"),t.style.setProperty("max-height","calc(100vh - var(--header-height))"),t.style.setProperty("padding","2px"),t.style.setProperty("z-index","999"),t.style.setProperty("margin-left","-2px"),p(t,m(z,{get each(){return s()},children:(e,n)=>{const l=()=>{let c=`p-rel w100 ${u.cms_editable_card}`;return i()?.id===e.id&&(c+=` ${u.cms_editable_card_selected}`),c};return e.marginTop&&(e.marginTop=void 0),e.marginBottom&&(e.marginBottom=void 0),(()=>{var c=k();return c.$$click=d=>{d.stopPropagation(),f(e);const C=[];for(const _ of Z(e.type)?.params||[]){const T=typeof _=="number"?_:_[0],$={...P.get(T)},E=typeof _=="number"?"":_[1];E&&($.name=E),e[$.key]&&($.content=e[$.key]),C.push($)}console.log("params a renderizar::",C),b(C)},p(c,m(B,{mode:1,onAddSeccion:()=>y(n(),1),onMoveSeccion:()=>x(n(),1)}),null),p(c,m(q,{args:e,get type(){return e.type}}),null),p(c,m(B,{mode:2,onAddSeccion:()=>y(n(),2),onMoveSeccion:()=>x(n(),2)}),null),h(()=>v(c,l())),c})()}})),t})()]}})}const B=r=>{const s={};return r.mode===1&&(s.top="0"),r.mode===2&&(s.bottom="0"),(()=>{var o=U(),i=o.firstChild,f=i.nextSibling;return F(o,s),i.$$click=a=>{a.stopPropagation(),r.onAddSeccion()},f.$$click=a=>{a.stopPropagation(),r.onMoveSeccion()},p(f,(()=>{var a=g(()=>r.mode===1);return()=>a()&&X()})(),null),p(f,(()=>{var a=g(()=>r.mode===2);return()=>a()&&Y()})(),null),h(a=>{var b=`flex p-abs ${u.cms_card_buttons}`,P=`flex ai-center p-rel ff-bold mr-06 ${u.cms_card_button}`,y=`flex ai-center p-rel ff-bold mr-06 ${u.cms_card_button} ${u.cs1}`;return b!==a.e&&v(o,a.e=b),P!==a.t&&v(i,a.t=P),y!==a.a&&v(f,a.a=y),a},{e:void 0,t:void 0,a:void 0}),o})()},I=r=>{const s=()=>{let o=`mt-04 mb-04 ${u.seccion_card}`;return r.isSelected&&(o+=" selected"),o};return(()=>{var o=k();return o.$$click=i=>{i.stopPropagation(),r.onSelect()},p(o,()=>r.params.name),h(()=>v(o,s())),o})()};D(["click"]);export{oe as default,ne as pageViews,ae as setPageViews};