import{a as p,f as ie,c as s,d as ce,i as n,S as b,b as A,e as P,t as m}from"./web-D2t8vN-4.js";import{c as de,t as ue,F as me,H as B,N as x,f as k,a as u,S as _,I as y,o as fe,r as D,P as ve,L as S,e as R}from"./app-De9iPHfG.js";import{s as C,M as V}from"./Modals-B-lhZTDN.js";import{Q as L}from"./QTable-CpfVE6kU.js";import{u as he,c as G,g as ge,a as pe,b as q,p as be,d as Ce,e as Se}from"./ventas-DYkryNFQ.js";import{u as je}from"./sedes-almacenes-PyW2IIXn.js";var we=m('<div class=lh-10><div class="ff-bold h3"></div><div class="h5 c-steel">'),$e=m('<div class="lh-10 t-r"><div class="ff-mono h5"></div><div class="h5 c-steel">'),xe=m("<div>"),z=m('<div class="box-error-ms mt-08">Seleccione una Caja'),_e=m('<div class="flex w100 jc-between mt-08"><div class="flex ai-center"><div class="h3 ff-bold mr-08"></div><button class="bn1 b-yellow"><i class=icon-pencil></i></button></div><div class="flex ai-center"><button class="bn1 b-green"><i class=icon-plus></i></button>'),ye=m('<div class="flex w100 jc-between mt-08"><div></div><div class="flex ai-center"><button class="bn1 b-green"><i class=icon-plus></i></button>'),De=m('<div class="flex jc-between mb-06"><div class=""><div class="flex jc-between w100 mb-10"><div class="search-c4 mr-16 w16rem"><div><i class=icon-search></i></div><input class=w100 autocomplete=off type=text></div><div class="flex ai-center"><button class="bn1 b-green"><i class=icon-plus></i></button></div></div></div><div class=card-c4>'),Ie=m('<div class="w100-10 flex-wrap in-s2"><div class="w100 flex jc-between ai-center"><div>'),Me=m('<div class="w100 c-red ff-bold"><i class=icon-attention>'),Te=m('<div class="w100-10 flex-wrap in-s2"><div class=w-10x><button class="bn1 b-purple"><i class="h5 icon-arrows-cw"></i>Recalcular</button></div><div class=w-10x>'),Ne=m('<div class="w100-10 flex-wrap in-s2"><div class=w-12x>'),H=m("<span>");function Le(){const U=de(G,"id"),[Q]=je(),[K,J]=p(""),[I,W]=p(1),[l,$]=p({}),[h,M]=p({}),[v,T]=p({}),[X,Y]=p([]),[Z,ee]=p([]),[c,N]=he(),ae=[{header:"ID",headerStyle:{width:"2rem"},css:"t-c c-purple2",getValue:a=>a.ID},{header:"Nombre",field:"Nombre",cellStyle:{"padding-top":"2px","padding-bottom":"3px"},render:a=>(()=>{var t=we(),r=t.firstChild,o=r.nextSibling;return n(r,()=>a.Nombre),n(o,()=>q.find(f=>f.id===a.Tipo)?.name||"-"),t})()},{header:"Cuadre",field:"Nombre",render:a=>a.CuadreFecha?(()=>{var t=$e(),r=t.firstChild,o=r.nextSibling;return n(r,()=>u(a.CuadreSaldo/100,2)),n(o,()=>k(a.CuadreFecha,"d-M h:n")),t})():""},{header:"Saldo",field:"Nombre",css:"t-r ff-mono",render:a=>(()=>{var t=xe();return n(t,()=>u(a.SaldoCurrent/100,2)),t})()}],te=async()=>{const a=l();if(!a.Nombre||!a.Tipo||!a.SedeID){x.failure("Los inputs Nombre, Tipo y Sede son obligatorios");return}S.standard("Guardando caja...");try{await be(a)}catch(t){console.warn(t);return}S.remove(),Object.assign(c().CajasMap.get(a.ID),a),c().Cajas=[...c().Cajas],N({...c()}),C([])},re=async()=>{const a=h();a.SaldoSistema=l().SaldoCurrent,S.standard("Guardando caja...");let t;try{t=await Ce(a)}catch(o){console.warn(o);return}S.remove();const r=c().CajasMap.get(a.CajaID);if(typeof t?.NeedUpdateSaldo=="number"){r.SaldoCurrent=t.NeedUpdateSaldo,$({...r});const o={...h()};o._error=`Hubo una actualización en el saldo de la caja. El saldo actual es "${u(r.SaldoCurrent/100),2}". Intente nuevamente con el cálculo actualizado.`,o.SaldoDiferencia=o.SaldoReal-r.SaldoCurrent,M(o)}else r.SaldoCurrent=a.SaldoReal,c().Cajas=[...c().Cajas],N({...c()}),Object.assign(l(),r),C([])},se=async()=>{const a=v();(!a.Tipo||!a.Monto)&&x.failure("Se necesita seleccionar un monto y un tipo."),S.standard("Guardando Movimiento...");let t;try{t=await Se(a)}catch(o){console.warn(o);return}S.remove();const r=c().CajasMap.get(a.CajaID);r.SaldoCurrent=a.SaldoFinal,c().Cajas=[...c().Cajas],N({...c()}),Object.assign(l(),r),C([])},E=ie(()=>[3].includes(v().Tipo));return s(ve,{title:"Cajas & Bancos",get children(){return[(()=>{var a=De(),t=a.firstChild,r=t.firstChild,o=r.firstChild,f=o.firstChild,j=f.nextSibling,ne=o.nextSibling,le=ne.firstChild,w=t.nextSibling;return j.$$keyup=e=>{e.stopPropagation(),ue(()=>{J((e.target.value||"").toLowerCase().trim())},150)},le.$$click=e=>{e.stopPropagation(),$({ID:-1,ss:1}),C([1])},n(t,s(L,{css:"selectable w100",columns:ae,maxHeight:"calc(100vh - 8rem - 16px)",styleMobile:{height:"100vh"},get data(){return c()?.Cajas||[]},get selected(){return l().ID},isSelected:(e,i)=>e?.ID===i,get filterText(){return K()},filterKeys:["nombre"],onRowCLick:e=>{const i=l().ID===e.ID?{}:{...e};$(i)}}),null),n(w,s(me,{get selectedID(){return I()},class:"w100",options:[[1,"Movimientos"],[2,"Cuadres"]],buttonStyle:{"min-height":"2.1rem"},buttonClass:"ff-bold",onSelect:e=>{W(e)}}),null),n(w,s(b,{get when(){return I()===1},get children(){return s(B,{get baseObject(){return l()},startPromise:async e=>{if(!e.ID)return;let i;try{i=await ge({CajaID:e.ID,lastRegistros:200})}catch(d){x.failure(d);return}Y(i)},get children(){return[s(b,{get when(){return!l().ID},get children(){return z()}}),s(b,{get when(){return!!l().ID},get children(){return[(()=>{var e=_e(),i=e.firstChild,d=i.firstChild,g=d.nextSibling,F=i.nextSibling,oe=F.firstChild;return n(d,()=>l()?.Nombre||""),g.$$click=O=>{O.stopPropagation(),C([1])},oe.$$click=O=>{O.stopPropagation(),C([3]),T({CajaID:l().ID,SaldoFinal:l().SaldoCurrent})},e})(),s(L,{css:"w100 mt-08",maxHeight:"calc(100vh - 8rem - 16px)",styleMobile:{height:"100vh"},get data(){return X()},columns:[{header:"Fecha Hora",getValue:e=>k(e.Created,"d-M h:n")},{header:"Tipo Mov.",getValue:e=>U.get(e.Tipo)?.name||""},{header:"Monto",css:"ff-mono t-r",render:e=>(()=>{var i=H();return n(i,()=>u(e.Monto/100,2)),A(()=>P(i,e.Monto<0?"c-red":"")),i})()},{header:"Saldo Final",css:"ff-mono t-r",getValue:e=>u(e.SaldoFinal/100,2)},{header:"Nº Documento",getValue:e=>""},{header:"Usuario",css:"t-c",getValue:e=>e.Usuario?.usuario||""}]})]}})]}})}}),null),n(w,s(b,{get when(){return I()===2},get children(){return s(B,{get baseObject(){return l()},startPromise:async e=>{if(!e.ID)return;let i;try{i=await pe({CajaID:e.ID,lastRegistros:200})}catch(d){x.failure(d);return}ee(i)},get children(){return[s(b,{get when(){return!l().ID},get children(){return z()}}),s(b,{get when(){return!!l().ID},get children(){return[(()=>{var e=ye(),i=e.firstChild,d=i.nextSibling,g=d.firstChild;return g.$$click=F=>{F.stopPropagation(),C([2]),M({CajaID:l().ID})},e})(),s(L,{css:"w100 mt-08",maxHeight:"calc(100vh - 8rem - 16px)",styleMobile:{height:"100vh"},get data(){return Z()},onRowCLick:e=>{const i=l().ID===e.ID?{}:{...e};$(i)},columns:[{header:"Fecha Hora",getValue:e=>k(e.Created,"d-M h:n")},{header:"Saldo Sistema",css:"ff-mono t-r",getValue:e=>u((e.SaldoSistema||0)/100,2)},{header:"Diferencia",css:"ff-mono t-r",getValue:e=>u((e.SaldoDiferencia||0)/100,2)},{header:"Saldo Real",css:"ff-mono t-r",getValue:e=>u((e.SaldoReal||0)/100,2)},{header:"Usuario",css:"t-c",getValue:e=>e.Usuario?.usuario||""}]})]}})]}})}}),null),A(e=>{var i=!![2,3].includes(R()),d=R()===1?"36%":"100%",g=R()===1?"calc(62.5% + 1rem)":"100%";return i!==e.e&&a.classList.toggle("column",e.e=i),d!==e.t&&((e.t=d)!=null?t.style.setProperty("width",d):t.style.removeProperty("width")),g!==e.a&&((e.a=g)!=null?w.style.setProperty("width",g):w.style.removeProperty("width")),e},{e:void 0,t:void 0,a:void 0}),a})(),s(V,{id:1,title:"Cajas",onSave:()=>{te()},onDelete:()=>{},get children(){var a=Ie(),t=a.firstChild;return t.firstChild,n(a,s(_,{get saveOn(){return l()},save:"Tipo",css:"w-10x mb-10",label:"Tipo",keys:"id.name",options:q,placeholder:"",required:!0}),t),n(a,s(y,{get saveOn(){return l()},save:"Nombre",css:"w-14x mb-10",label:"Nombre",required:!0}),t),n(a,s(y,{get saveOn(){return l()},save:"Descripcion",css:"w-24x mb-10",label:"Descripcion"}),t),n(a,s(_,{get saveOn(){return l()},save:"SedeID",css:"w-10x mb-10",label:"Sede",required:!0,get options(){return Q().Sedes},keys:"ID.Nombre"}),t),n(t,s(fe,{label:"Saldo Negativo",get saveOn(){return l()},save:"Nombre"}),null),a}}),s(V,{id:2,title:"Cuadre de Caja",onSave:()=>{re()},get children(){var a=Te(),t=a.firstChild,r=t.firstChild,o=t.nextSibling;return n(a,s(D,{css:"w-14x mb-10",label:"Saldo Sistema",inputCss:"h3 ff-mono jc-center",get content(){return u(l().SaldoCurrent/100,2)}}),t),r.style.setProperty("margin-top","0.9rem"),n(a,s(y,{get saveOn(){return h()},save:"SaldoReal",type:"number",inputCss:"h3 ff-mono t-c",baseDecimals:2,css:"w-14x mb-10",label:"Saldo Encontrado",required:!0,onChange:()=>{console.log("caja cuadre::",h()),M({...h()})}}),o),n(a,s(D,{css:"w-14x mb-10",label:"Diferencia",inputCss:"h3 ff-mono jc-center",getContent:()=>{const f=(h().SaldoReal||0)-l().SaldoCurrent;return f?(()=>{var j=H();return P(j,f>0?"c-blue":"c-red"),n(j,()=>u(f/100,2)),j})():""}}),null),n(a,s(b,{get when(){return h()._error},get children(){var f=Me();return f.firstChild,n(f,()=>h()._error,null),f}}),null),a}}),s(V,{id:3,title:"Movimiento de Caja",onSave:()=>{se()},get children(){var a=Ne(),t=a.firstChild;return n(a,s(_,{get saveOn(){return v()},save:"Tipo",css:"w-12x mb-10",label:"Tipo",keys:"id.name",get options(){return G.filter(r=>r.group===2)},placeholder:"",required:!0,onChange:()=>{v().CajaRefID=0,T({...v()})}}),t),n(a,s(_,{get saveOn(){return v()},save:"CajaRefID",css:"w-12x mb-10",label:"Caja Destino",keys:"id.name",options:q,get disabled(){return!E()},get placeholder(){return E()?"seleccione":"no aplica"},required:!0}),t),n(a,s(y,{get saveOn(){return v()},save:"Monto",inputCss:"ff-mono h3 t-c",css:"w-12x mb-10",label:"Monto",baseDecimals:2,required:!0,type:"number",transform:r=>{const o=U.get(v().Tipo);return console.log("movimiento tipo::",o),o?.isNegative&&typeof r=="number"&&r>0&&(r=r*-1),r},onChange:()=>{const r={...v()};r.SaldoFinal=l().SaldoCurrent+(r.Monto||0),T(r)}}),t),n(a,s(D,{css:"w-12x mb-10",label:"Saldo Inicial",inputCss:"h3 ff-mono jc-center",get content(){return u(l().SaldoCurrent/100,2)}}),null),n(a,s(D,{css:"w-12x mb-10",label:"Saldo Final",inputCss:"h3 ff-mono jc-center",getContent:()=>{const r=v().SaldoFinal;return(()=>{var o=H();return P(o,r>=0?"":"c-red"),n(o,()=>u(r/100,2)),o})()}}),null),a}})]}})}ce(["keyup","click"]);export{Le as default};