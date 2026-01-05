import { M as append, P as from_html, Q as sibling, W as template_effect, X as child, Z as first_child, _t as reset, j as set_text, lt as pop, u as init, ut as push } from "../chunks/CTB4HzdN.js";
import "../chunks/D25ese0K.js";
import "../chunks/2yAUi6s0.js";
import { t as page } from "../chunks/Dm_jeqTV.js";
import "../chunks/BGbcPCgU.js";
var root = from_html(`<h1> </h1> <p> </p>`, 1);
function Error($$anchor, $$props) {
	push($$props, false);
	init();
	var fragment = root();
	var h1 = first_child(fragment);
	var text = child(h1, true);
	reset(h1);
	var p = sibling(h1, 2);
	var text_1 = child(p, true);
	reset(p);
	template_effect(() => {
		set_text(text, page.status);
		set_text(text_1, page.error?.message);
	});
	append($$anchor, fragment);
	pop();
}
export { Error as component };
