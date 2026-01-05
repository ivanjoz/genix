import { A as if_block, E as snippet, G as user_effect, M as append, N as comment, P as from_html, Q as sibling, X as child, Z as first_child, _t as reset, lt as pop, n as onDestroy, r as onMount, st as user_derived, ut as push, z as get } from "./CTB4HzdN.js";
import { d as checkIsLogin, r as closeAllModals, t as Core, y as Env } from "./BwrZ3UQO.js";
var root_2 = from_html(`<div class="p-16"><h2>Cargando...</h2></div>`);
var root = from_html(`<div class="_1 p-10 aMj"><!> <!></div>`);
function Page($$anchor, $$props) {
	push($$props, true);
	Env.sideLayerSize = $$props.sideLayerSize || 0;
	const isLogged = user_derived(() => checkIsLogin() === 2);
	user_effect(() => {
		console.log("isLogged 22", get(isLogged));
	});
	onMount(() => {
		console.log("isLogged 11", get(isLogged));
		if (!get(isLogged)) Env.navigate("/login");
		else {
			Core.pageTitle = $$props.title || "";
			Core.pageOptions = $$props.options || [];
		}
	});
	onDestroy(() => {
		Core.openSideLayer(0);
		closeAllModals();
	});
	var div = root();
	var node = child(div);
	var consequent = ($$anchor$1) => {
		var fragment = comment();
		snippet(first_child(fragment), () => $$props.children);
		append($$anchor$1, fragment);
	};
	if_block(node, ($$render) => {
		if (Core.isLoading === 0 && get(isLogged)) $$render(consequent);
	});
	var node_2 = sibling(node, 2);
	var consequent_1 = ($$anchor$1) => {
		append($$anchor$1, root_2());
	};
	if_block(node_2, ($$render) => {
		if (Core.isLoading > 0) $$render(consequent_1);
	});
	reset(div);
	append($$anchor, div);
	pop();
}
export { Page as t };
