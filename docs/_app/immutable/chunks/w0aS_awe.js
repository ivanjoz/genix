import { A as if_block, G as user_effect, L as delegate, M as append, P as from_html, Q as sibling, S as set_class, V as untrack, W as template_effect, X as child, _t as reset, a as prop, at as state, j as set_text, lt as pop, rt as set, ut as push, v as set_attribute, z as get } from "./CTB4HzdN.js";
var root_1 = from_html(`<i class="icon-ok"></i>`);
var root = from_html(`<div><button><!></button> <label> </label></div>`);
function Checkbox($$anchor, $$props) {
	push($$props, true);
	const saveOn = prop($$props, "saveOn", 15);
	let isSelected = state(false);
	const onSelect = () => {
		set(isSelected, !get(isSelected));
		if (saveOn() && $$props.save) saveOn(saveOn()[$$props.save] = get(isSelected), true);
	};
	let lastSaveOn;
	user_effect(() => {
		if (!saveOn() || !$$props.save) return;
		if (lastSaveOn === saveOn()) return;
		lastSaveOn = saveOn();
		untrack(() => {
			set(isSelected, !!saveOn()[$$props.save]);
		});
	});
	var div = root();
	var button = child(div);
	let classes;
	button.__click = (ev) => {
		onSelect();
	};
	var node = child(button);
	var consequent = ($$anchor$1) => {
		append($$anchor$1, root_1());
	};
	if_block(node, ($$render) => {
		if (get(isSelected)) $$render(consequent);
	});
	reset(button);
	var label_1 = sibling(button, 2);
	var text = child(label_1, true);
	reset(label_1);
	reset(div);
	template_effect(() => {
		set_class(div, 1, `flex items-center ${$$props.css ?? ""}`, "aMs");
		classes = set_class(button, 1, "flex mr-4 pt-1 items-center p-0 lh-10 justify-center rounded-[4px] shrink-0 w-28 h-26 _1 aMs", null, classes, { _2: get(isSelected) });
		set_attribute(button, "aria-label", $$props.label);
		set_text(text, $$props.label);
	});
	append($$anchor, div);
	pop();
}
delegate(["click"]);
export { Checkbox as t };
