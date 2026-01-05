const __vite__mapDeps=(i,m=__vite__mapDeps,d=(m.f||(m.f=["../nodes/0.B7egm1ly.js","../chunks/D9PGjZ6Y.js","../chunks/D25ese0K.js","../chunks/CTB4HzdN.js","../chunks/BemFyt0E.js","../chunks/BwrZ3UQO.js","../chunks/DwKmqXYU.js","../chunks/Dm_jeqTV.js","../chunks/2yAUi6s0.js","../chunks/B0DNpKVT.js","../chunks/N4ddDIuf.js","../chunks/cyfMRGwx.js","../assets/components.CtDQGghW.css","../chunks/GeoukCMt.js","../assets/Page._IUKH2t9.css","../chunks/CGFlWCTf.js","../assets/OptionsStrip.0Ds6k4y3.css","../chunks/CoH7Ebne.js","../chunks/DOFkf9MZ.js","../chunks/BKsRj9OC.js","../assets/0.DXLt86_H.css","../nodes/1.BJNEKsgY.js","../chunks/BGbcPCgU.js","../nodes/2.CGAhRYVH.js","../nodes/3.Clg-ZHLr.js","../chunks/BCJF-hdW.js","../chunks/DnSz4jvr.js","../assets/popover2.CmFlxH-O.css","../assets/vTable.CdrdVPWq.css","../assets/3.BijVyY_o.css","../nodes/4.DMB8zaS2.js","../chunks/iQLQpuQM.js","../assets/Modal.CY5vNf7-.css","../nodes/5.DGuOdnOP.js","../nodes/6.D48cWAQB.js","../chunks/Bmy_fcV9.js","../assets/CheckboxOptions.Cxs5OOIt.css","../chunks/C1AypaHL.js","../assets/Layer.3IKOlC54.css","../chunks/BHNEJas0.js","../chunks/BOmyLNB5.js","../assets/SearchSelect.DWFlXYQL.css","../assets/SearchCard.Cab6UEHr.css","../assets/6.en6_9a5I.css","../nodes/7.z_WHG9mT.js","../nodes/8.CXmTBBJM.js","../nodes/9.DLzhyGiD.js","../chunks/DbHvoCmI.js","../assets/Popover2Example.bI5FR2mI.css","../assets/popover2.B6DPw75D.css","../nodes/10.BtJIyGdD.js","../nodes/11.CvwhtK9N.js","../assets/11.CQXSR9v_.css","../nodes/12.XtP11gmL.js","../chunks/DoSytdLd.js","../assets/12.B_6WUl9M.css","../nodes/13.4CVDrD4i.js","../assets/13.CVY7wd7p.css","../nodes/14.VLN1HHHh.js","../assets/14.DL0hDo4p.css","../nodes/15.C1_7ZslS.js","../chunks/5nFXSVM4.js","../assets/ImageUploader.DX-Erxjf.css","../nodes/16.BRDLYm4J.js","../chunks/o-eLsTb7.js","../assets/16.CUJlK1YT.css","../nodes/17.DOjUwDJn.js","../assets/17.D_mEJ8dz.css","../nodes/18.DNU2PS8E.js","../chunks/ArQKZ6r_.js","../assets/DateInput.oggMX8jg.css","../chunks/DAmFAq63.js","../chunks/BVhuLbLV.js","../nodes/19.CUugcCqd.js","../chunks/w0aS_awe.js","../assets/Checkbox.BOcRoMSM.css","../chunks/DBgcaFpj.js","../nodes/20.B0LuFpUW.js","../nodes/21.CY_wY0F-.js","../assets/21.C0fJmso2.css","../nodes/22.CJAanByv.js","../nodes/23.D7ccUr3g.js","../assets/23.DwtGuj1N.css"])))=>i.map(i=>d[i]);
import { A as if_block, B as tick, F as text, G as user_effect, K as user_pre_effect, M as append, N as comment, P as from_html, Q as sibling, T as component, W as template_effect, X as child, Z as first_child, _t as reset, a as prop, at as state, dt as setContext, f as bind_this, i as asClassComponent, j as set_text, lt as pop, r as onMount, rt as set, st as user_derived, ut as push, z as get } from "../chunks/CTB4HzdN.js";
import { t as __vitePreload } from "../chunks/DoSytdLd.js";
import "../chunks/2yAUi6s0.js";
import { t as browser } from "../chunks/BemFyt0E.js";
const matchers = {};
var root_4 = from_html(`<div id="svelte-announcer" aria-live="assertive" aria-atomic="true" style="position: absolute; left: 0; top: 0; clip: rect(0 0 0 0); clip-path: inset(50%); overflow: hidden; white-space: nowrap; width: 1px; height: 1px"><!></div>`);
var root = from_html(`<!> <!>`, 1);
function Root($$anchor, $$props) {
	push($$props, true);
	let components = prop($$props, "components", 23, () => []), data_0 = prop($$props, "data_0", 3, null), data_1 = prop($$props, "data_1", 3, null);
	if (!browser) setContext("__svelte__", $$props.stores);
	if (browser) user_pre_effect(() => $$props.stores.page.set($$props.page));
	else $$props.stores.page.set($$props.page);
	user_effect(() => {
		$$props.stores;
		$$props.page;
		$$props.constructors;
		components();
		$$props.form;
		data_0();
		data_1();
		$$props.stores.page.notify();
	});
	let mounted = state(false);
	let navigated = state(false);
	let title = state(null);
	onMount(() => {
		const unsubscribe = $$props.stores.page.subscribe(() => {
			if (get(mounted)) {
				set(navigated, true);
				tick().then(() => {
					set(title, document.title || "untitled page", true);
				});
			}
		});
		set(mounted, true);
		return unsubscribe;
	});
	const Pyramid_1 = user_derived(() => $$props.constructors[1]);
	var fragment = root();
	var node = first_child(fragment);
	var consequent = ($$anchor$1) => {
		const Pyramid_0 = user_derived(() => $$props.constructors[0]);
		var fragment_1 = comment();
		component(first_child(fragment_1), () => get(Pyramid_0), ($$anchor$2, Pyramid_0_1) => {
			bind_this(Pyramid_0_1($$anchor$2, {
				get data() {
					return data_0();
				},
				get form() {
					return $$props.form;
				},
				get params() {
					return $$props.page.params;
				},
				children: ($$anchor$3, $$slotProps) => {
					var fragment_2 = comment();
					component(first_child(fragment_2), () => get(Pyramid_1), ($$anchor$4, Pyramid_1_1) => {
						bind_this(Pyramid_1_1($$anchor$4, {
							get data() {
								return data_1();
							},
							get form() {
								return $$props.form;
							},
							get params() {
								return $$props.page.params;
							}
						}), ($$value) => components()[1] = $$value, () => components()?.[1]);
					});
					append($$anchor$3, fragment_2);
				},
				$$slots: { default: true }
			}), ($$value) => components()[0] = $$value, () => components()?.[0]);
		});
		append($$anchor$1, fragment_1);
	};
	var alternate = ($$anchor$1) => {
		const Pyramid_0 = user_derived(() => $$props.constructors[0]);
		var fragment_3 = comment();
		component(first_child(fragment_3), () => get(Pyramid_0), ($$anchor$2, Pyramid_0_2) => {
			bind_this(Pyramid_0_2($$anchor$2, {
				get data() {
					return data_0();
				},
				get form() {
					return $$props.form;
				},
				get params() {
					return $$props.page.params;
				}
			}), ($$value) => components()[0] = $$value, () => components()?.[0]);
		});
		append($$anchor$1, fragment_3);
	};
	if_block(node, ($$render) => {
		if ($$props.constructors[1]) $$render(consequent);
		else $$render(alternate, false);
	});
	var node_4 = sibling(node, 2);
	var consequent_2 = ($$anchor$1) => {
		var div = root_4();
		var node_5 = child(div);
		var consequent_1 = ($$anchor$2) => {
			var text$1 = text();
			template_effect(() => set_text(text$1, get(title)));
			append($$anchor$2, text$1);
		};
		if_block(node_5, ($$render) => {
			if (get(navigated)) $$render(consequent_1);
		});
		reset(div);
		append($$anchor$1, div);
	};
	if_block(node_4, ($$render) => {
		if (get(mounted)) $$render(consequent_2);
	});
	append($$anchor, fragment);
	pop();
}
var root_default = asClassComponent(Root);
const nodes = [
	() => __vitePreload(() => import("../nodes/0.B7egm1ly.js"), __vite__mapDeps([0,1,2,3,4,5,6,7,8,9,10,11,12,13,14,15,16,17,18,19,20]), import.meta.url),
	() => __vitePreload(() => import("../nodes/1.BJNEKsgY.js"), __vite__mapDeps([21,2,1,3,7,8,22]), import.meta.url),
	() => __vitePreload(() => import("../nodes/2.CGAhRYVH.js"), __vite__mapDeps([23,3,8,22]), import.meta.url),
	() => __vitePreload(() => import("../nodes/3.Clg-ZHLr.js"), __vite__mapDeps([24,25,1,5,2,3,4,6,26,9,27,28,8,13,14,18,29]), import.meta.url),
	() => __vitePreload(() => import("../nodes/4.DMB8zaS2.js"), __vite__mapDeps([30,25,1,5,2,3,4,6,26,9,27,28,8,10,11,12,31,32,13,14,18]), import.meta.url),
	() => __vitePreload(() => import("../nodes/5.DGuOdnOP.js"), __vite__mapDeps([33,2,1,3,5,4,6,8,22,10,11,12,13,14]), import.meta.url),
	() => __vitePreload(() => import("../nodes/6.D48cWAQB.js"), __vite__mapDeps([34,25,1,5,2,3,4,6,26,9,27,28,8,35,36,10,11,12,37,15,16,38,31,32,13,14,39,40,18,41,42,17,43]), import.meta.url),
	() => __vitePreload(() => import("../nodes/7.z_WHG9mT.js"), __vite__mapDeps([44,25,1,5,2,3,4,6,26,9,27,28,8,10,11,12,31,32,13,14,39,40,18,41,42,19]), import.meta.url),
	() => __vitePreload(() => import("../nodes/8.CXmTBBJM.js"), __vite__mapDeps([45,2,1,3,5,4,6,8,22,13,14]), import.meta.url),
	() => __vitePreload(() => import("../nodes/9.DLzhyGiD.js"), __vite__mapDeps([46,3,6,8,22,26,9,27,47,48,49]), import.meta.url),
	() => __vitePreload(() => import("../nodes/10.BtJIyGdD.js"), __vite__mapDeps([50,3,6,8,22,26,9,27,47,48,49]), import.meta.url),
	() => __vitePreload(() => import("../nodes/11.CvwhtK9N.js"), __vite__mapDeps([51,3,6,8,26,9,27,52,49]), import.meta.url),
	() => __vitePreload(() => import("../nodes/12.XtP11gmL.js"), __vite__mapDeps([53,54,3,8,22,55]), import.meta.url),
	() => __vitePreload(() => import("../nodes/13.4CVDrD4i.js"), __vite__mapDeps([56,3,8,57]), import.meta.url),
	() => __vitePreload(() => import("../nodes/14.VLN1HHHh.js"), __vite__mapDeps([58,3,8,59]), import.meta.url),
	() => __vitePreload(() => import("../nodes/15.C1_7ZslS.js"), __vite__mapDeps([60,2,1,3,5,4,6,8,22,61,18,62,13,14]), import.meta.url),
	() => __vitePreload(() => import("../nodes/16.BRDLYm4J.js"), __vite__mapDeps([63,25,1,5,2,3,4,6,26,9,27,28,8,13,14,64,65]), import.meta.url),
	() => __vitePreload(() => import("../nodes/17.DOjUwDJn.js"), __vite__mapDeps([66,2,1,3,5,4,6,8,10,11,12,67]), import.meta.url),
	() => __vitePreload(() => import("../nodes/18.DNU2PS8E.js"), __vite__mapDeps([68,25,1,5,2,3,4,6,26,9,27,28,8,69,11,12,70,13,14,40,18,41,71,72]), import.meta.url),
	() => __vitePreload(() => import("../nodes/19.CUugcCqd.js"), __vite__mapDeps([73,25,1,5,2,3,4,6,26,9,27,28,8,74,75,10,11,12,31,32,13,14,40,18,41,15,16,76,72]), import.meta.url),
	() => __vitePreload(() => import("../nodes/20.B0LuFpUW.js"), __vite__mapDeps([77,25,1,5,2,3,4,6,26,9,27,28,8,69,11,12,70,13,14,40,18,41,76]), import.meta.url),
	() => __vitePreload(() => import("../nodes/21.CY_wY0F-.js"), __vite__mapDeps([78,25,1,5,2,3,4,6,26,9,27,28,8,35,36,61,18,62,10,11,12,37,15,16,38,31,32,13,14,39,40,41,42,64,71,79]), import.meta.url),
	() => __vitePreload(() => import("../nodes/22.CJAanByv.js"), __vite__mapDeps([80,25,1,5,2,3,4,6,26,9,27,28,8,74,75,10,11,12,37,15,16,38,13,14,40,18,41,71,72]), import.meta.url),
	() => __vitePreload(() => import("../nodes/23.D7ccUr3g.js"), __vite__mapDeps([81,25,1,5,2,3,4,6,26,9,27,28,8,10,11,12,37,15,16,38,31,32,13,14,40,18,41,72,82]), import.meta.url)
];
const server_loads = [];
const dictionary = {
	"/": [2],
	"/admin/backups": [3],
	"/admin/empresas": [4],
	"/admin/parametros": [5],
	"/admin/perfiles-accesos": [6],
	"/admin/usuarios": [7],
	"/cms/inicio": [8],
	"/develop-ui/demo1": [9],
	"/develop-ui/demo2-overflow": [11],
	"/develop-ui/demo2": [10],
	"/develop-ui/model-test2": [13],
	"/develop-ui/model-test3": [14],
	"/develop-ui/model-test": [12],
	"/develop-ui/test-cards": [15],
	"/develop-ui/test-table": [16],
	"/login": [17],
	"/operaciones/almacen-movimientos": [18],
	"/operaciones/cajas-movimientos": [20],
	"/operaciones/cajas": [19],
	"/operaciones/productos-stock": [22],
	"/operaciones/productos": [21],
	"/operaciones/sedes-almacenes": [23]
};
const hooks = {
	handleError: (({ error }) => {
		console.error(error);
	}),
	reroute: (() => {}),
	transport: {}
};
const decoders = Object.fromEntries(Object.entries(hooks.transport).map(([k, v]) => [k, v.decode]));
const hash = false;
const decode = (type, value) => decoders[type](value);
export { decode, decoders, dictionary, hash, hooks, matchers, nodes, root_default as root, server_loads };
