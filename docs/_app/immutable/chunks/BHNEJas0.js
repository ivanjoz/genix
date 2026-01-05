import { $ as proxy, C as clsx, G as user_effect, L as delegate, M as append, O as each, P as from_html, Q as sibling, S as set_class, V as untrack, W as template_effect, X as child, _t as reset, a as prop, at as state, j as set_text, k as index, lt as pop, rt as set, st as user_derived, ut as push, z as get } from "./CTB4HzdN.js";
import { n as WeakSearchRef } from "./BwrZ3UQO.js";
import { t as SearchSelect } from "./BOmyLNB5.js";
var root_1 = from_html(`<div class="m-2 px-8 min-w-56 h-32 lh-10 flex _3 aMl"> <button class="_4 absolute w-28 h-28 rounded right-2 aMl" aria-label="eliminar"><i class="icon-trash"></i></button></div>`);
var root = from_html(`<div><!> <div></div></div>`);
function SearchCard($$anchor, $$props) {
	push($$props, true);
	const saveOn = prop($$props, "saveOn", 15), css = prop($$props, "css", 3, ""), options = prop($$props, "options", 19, () => []);
	let selectedIDs = state(proxy([]));
	const buildMap = () => {
		if (WeakSearchRef.has(options())) return;
		const idToRecord = /* @__PURE__ */ new Map();
		const valueToRecord = /* @__PURE__ */ new Map();
		for (const e of options()) {
			idToRecord.set(e[$$props.keyId], e);
			valueToRecord.set(e[$$props.keyName].toLowerCase(), e);
		}
		WeakSearchRef.set(options(), {
			idToRecord,
			valueToRecord
		});
	};
	buildMap();
	user_effect(() => {
		if (options()?.length > 0) buildMap();
	});
	const doOnChange = () => {
		if (saveOn() && $$props.save) saveOn(saveOn()[$$props.save] = get(selectedIDs), true);
		if ($$props.onChange) $$props.onChange(get(selectedIDs));
	};
	let prevSaveOn;
	user_effect(() => {
		if (saveOn() && $$props.save && saveOn() !== prevSaveOn) {
			prevSaveOn = saveOn();
			buildMap();
			untrack(() => {
				set(selectedIDs, saveOn()[$$props.save] || [], true);
			});
		}
	});
	const getOption = (id) => {
		return (WeakSearchRef.get(options())?.idToRecord || /* @__PURE__ */ new Map()).get(id) || {
			[$$props.keyName]: `ID-${id}`,
			[$$props.keyId]: id
		};
	};
	var div = root();
	var node = child(div);
	{
		let $0 = user_derived(() => "s1 " + $$props.inputCss);
		SearchSelect(node, {
			get options() {
				return options();
			},
			get keyId() {
				return $$props.keyId;
			},
			get keyName() {
				return $$props.keyName;
			},
			clearOnSelect: true,
			get avoidIDs() {
				return get(selectedIDs);
			},
			get placeholder() {
				return $$props.label;
			},
			get css() {
				return get($0);
			},
			get optionsCss() {
				return $$props.optionsCss;
			},
			onChange: (e) => {
				if (!e) return;
				const id = e[$$props.keyId];
				if (!get(selectedIDs).includes(id)) {
					get(selectedIDs).push(id);
					doOnChange();
				}
			}
		});
	}
	var div_1 = sibling(node, 2);
	each(div_1, 21, () => get(selectedIDs), index, ($$anchor$1, id) => {
		const el = user_derived(() => getOption(get(id)));
		var div_2 = root_1();
		var text = child(div_2);
		var button = sibling(text);
		button.__click = (ev) => {
			ev.stopPropagation();
			set(selectedIDs, get(selectedIDs).filter((x) => x !== get(id)), true);
			doOnChange();
		};
		reset(div_2);
		template_effect(() => set_text(text, `${get(el)[$$props.keyName] ?? ""} `));
		append($$anchor$1, div_2);
	});
	reset(div_1);
	reset(div);
	template_effect(() => {
		set_class(div, 1, clsx(css()), "aMl");
		set_class(div_1, 1, `p-4 min-h-40 _2 flex flex-wrap ${$$props.cardCss ?? ""}`, "aMl");
	});
	append($$anchor, div);
	pop();
}
delegate(["click"]);
export { SearchCard as t };
