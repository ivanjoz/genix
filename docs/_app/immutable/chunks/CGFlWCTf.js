import { A as if_block, L as delegate, M as append, O as each, P as from_html, Q as sibling, S as set_class, W as template_effect, X as child, _t as reset, j as set_text, k as index, lt as pop, st as user_derived, ut as push, z as get } from "./CTB4HzdN.js";
import { t as Core } from "./BwrZ3UQO.js";
var root_2 = from_html(`<span> </span>`);
var root_3 = from_html(`<span> <br/> </span>`);
var root_1 = from_html(`<button><!></button>`);
var root = from_html(`<div></div>`);
function OptionsStrip($$anchor, $$props) {
	push($$props, true);
	const getClass = (e) => {
		let cn = "";
		if ((Array.isArray(e) ? e[0] : ($$props.keyId ? e?.[$$props.keyId] : 0) || 0) === $$props.selected) cn += " _3";
		if ($$props.buttonCss) cn += " " + $$props.buttonCss;
		return cn;
	};
	const getValue = (e) => {
		if (Array.isArray(e)) {
			if (Core.deviceType === 3 && Array.isArray(e[2])) return e[2];
			return [e[1]];
		} else if ($$props.keyName) return [e[$$props.keyName]];
		else return [""];
	};
	var div = root();
	let classes;
	each(div, 21, () => $$props.options, index, ($$anchor$1, opt) => {
		const words = user_derived(() => getValue(get(opt)));
		var button = root_1();
		button.__click = (ev) => {
			ev.stopPropagation();
			$$props.onSelect(get(opt));
		};
		var node = child(button);
		var consequent = ($$anchor$2) => {
			var span = root_2();
			var text = child(span, true);
			reset(span);
			template_effect(() => set_text(text, get(words)[0]));
			append($$anchor$2, span);
		};
		var alternate = ($$anchor$2) => {
			var span_1 = root_3();
			var text_1 = child(span_1, true);
			var text_2 = sibling(text_1, 2, true);
			reset(span_1);
			template_effect(() => {
				set_text(text_1, get(words)[0]);
				set_text(text_2, get(words)[1]);
			});
			append($$anchor$2, span_1);
		};
		if_block(node, ($$render) => {
			if (get(words).length === 1) $$render(consequent);
			else $$render(alternate, false);
		});
		reset(button);
		template_effect(($0) => set_class(button, 1, `flex items-center ff-bold _2 ${$0 ?? ""}`, "aMt"), [() => getClass(get(opt))]);
		append($$anchor$1, button);
	});
	reset(div);
	template_effect(() => classes = set_class(div, 1, `_1 pb-4 md:pb-0 flex items-center shrink-0 max-w-[100%] overflow-x-auto overflow-y-hidden ${$$props.css ?? ""}`, "aMt", classes, {
		_5: $$props.useMobileGrid,
		"grid-cols-2": $$props.useMobileGrid && $$props.options.length === 2,
		"grid-cols-3": $$props.useMobileGrid && $$props.options.length === 3,
		"grid-cols-4": $$props.useMobileGrid && $$props.options.length === 4,
		"grid-cols-5": $$props.useMobileGrid && $$props.options.length === 5
	}));
	append($$anchor, div);
	pop();
}
delegate(["click"]);
export { OptionsStrip as t };
