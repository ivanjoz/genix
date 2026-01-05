import { $ as proxy, A as if_block, F as text, L as delegate, M as append, N as comment, P as from_html, Q as sibling, W as template_effect, X as child, Z as first_child, _t as reset, at as state, j as set_text, lt as pop, rt as set, ut as push, vt as noop, z as get } from "../chunks/CTB4HzdN.js";
import "../chunks/D25ese0K.js";
import "../chunks/2yAUi6s0.js";
import "../chunks/BwrZ3UQO.js";
import "../chunks/DwKmqXYU.js";
import { t as Page } from "../chunks/GeoukCMt.js";
import { t as VTable } from "../chunks/BCJF-hdW.js";
import "../chunks/DnSz4jvr.js";
import "../chunks/o-eLsTb7.js";
var root_3 = from_html(`<span class="aLT aMi">Updated</span>`);
var root_2 = from_html(`<div class="aLS aLR aMi"><strong>Selected:</strong> <!></div>`);
var root_5 = from_html(`<span style="color: #ef4444; font-weight: 600;"> </span>`);
var root_7 = from_html(`<span style="background-color: #fef3c7; padding: 0.125rem 0.25rem; border-radius: 3px;"> </span>`);
var root_9 = from_html(`<div class="aM1"><button class="aM2 aM3" title="Edit">✏️</button> <button class="aM2 aM4" title="Delete">🗑️</button></div>`);
var root_1 = from_html(`<div class="aLL aMi"><h2 class="aMi">VTable Component - Snippet-based cellRenderer Demo</h2> <p class="aLM aLN aLR aMi"> </p> <p class="aLM aLO aLQ aMi">Using <strong>cellRendererSnippet</strong> to render actual Svelte components
      in cells!</p> <p class="aLM aLP aLR aMi">💡 <strong>Tip:</strong> Click any row to update it. Try the Edit/Delete buttons
      in the Actions column!</p> <!></div> <!>`, 1);
function _page($$anchor, $$props) {
	push($$props, true);
	let data = state(proxy(makeData()));
	let selectedRecord = state(void 0);
	const columns = [
		{
			header: "ID",
			id: 101,
			renderHTML: (e) => {
				if (e._updated) return `<span class="updated-indicator">🔄</span> ${e.id}`;
				return e.id;
			},
			css: "id-column"
		},
		{
			header: "Personal Information",
			headerCss: "text-center header-group",
			subcols: [{
				header: "Edad",
				getValue: (e) => e.edad,
				css: "text-center"
			}, {
				header: "Nombre",
				id: 103,
				getValue: (e) => e.nombre,
				css: "nombre-column"
			}]
		},
		{
			header: "Contact & Details",
			headerCss: "text-center header-group",
			subcols: [{
				header: "Apellidos",
				getValue: (e) => e.apellidos,
				onCellEdit(e, value) {
					e.apellidos = value;
				}
			}, {
				header: "Edad",
				getValue: (e) => e.edadChanged || e.edad,
				render: (e) => {
					if (e.edadChanged && e.edadChanged !== e.edad) return [{
						css: "flex items-center",
						children: [
							{ text: `${e.edad}` },
							{
								css: "icon-ok",
								text: ""
							},
							{ text: `${e.edadChanged}` }
						]
					}];
					return [{ text: `${e.edad} !!` }];
				},
				css: "text-center",
				onCellEdit(e, value) {
					e.edadChanged = parseInt(value);
				}
			}]
		},
		{
			header: "Actions",
			id: "actions",
			css: "text-center action-column",
			getValue: () => ""
		}
	];
	function makeData() {
		console.log("generando data::");
		const records = [];
		for (let i = 0; i < 15e3; i++) {
			const record = {
				id: makeid(12),
				edad: Math.floor(Math.random() * 100),
				nombre: makeid(18).toLowerCase(),
				apellidos: makeid(23),
				numero: Math.floor(Math.random() * 1e3),
				selector: ""
			};
			records.push(record);
		}
		return records;
	}
	function makeid(length) {
		let result = "";
		const characters = "ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789";
		const charactersLength = 62;
		let counter = 0;
		while (counter < length) {
			result += characters.charAt(Math.floor(Math.random() * charactersLength));
			counter += 1;
		}
		return result;
	}
	function handleRowClick(record, index) {
		console.log("Row clicked:", record, "at index:", index);
		record.nombre = record.nombre + "_1";
		record._updated = !record._updated;
		set(selectedRecord, record, true);
		set(data, [...get(data)], true);
	}
	function isRowSelected(record, selected) {
		return record === selected;
	}
	function handleEdit(record) {
		console.log("Edit:", record);
		alert(`Editing: ${record.nombre}`);
	}
	function handleDelete(record) {
		if (confirm(`Delete ${record.nombre}?`)) set(data, get(data).filter((e) => e !== record), true);
	}
	Page($$anchor, {
		children: ($$anchor$1, $$slotProps) => {
			var fragment_1 = root_1();
			var div = first_child(fragment_1);
			var p = sibling(child(div), 2);
			var text$1 = child(p);
			reset(p);
			var node = sibling(p, 6);
			var consequent_1 = ($$anchor$2) => {
				var div_1 = root_2();
				var text_1 = sibling(child(div_1));
				var node_1 = sibling(text_1);
				var consequent = ($$anchor$3) => {
					append($$anchor$3, root_3());
				};
				if_block(node_1, ($$render) => {
					if (get(selectedRecord)._updated) $$render(consequent);
				});
				reset(div_1);
				template_effect(() => set_text(text_1, ` ${get(selectedRecord).nombre ?? ""} (ID: ${get(selectedRecord).id ?? ""}) `));
				append($$anchor$2, div_1);
			};
			if_block(node, ($$render) => {
				if (get(selectedRecord)) $$render(consequent_1);
			});
			reset(div);
			var node_2 = sibling(div, 2);
			{
				const cellRenderer = ($$anchor$2, record = noop, col = noop, value = noop) => {
					var fragment_2 = comment();
					var node_3 = first_child(fragment_2);
					var consequent_2 = ($$anchor$3) => {
						var span_1 = root_5();
						var text_2 = child(span_1);
						reset(span_1);
						template_effect(() => set_text(text_2, `${value() ?? ""} 🔥`));
						append($$anchor$3, span_1);
					};
					var alternate_2 = ($$anchor$3) => {
						var fragment_3 = comment();
						var node_4 = first_child(fragment_3);
						var consequent_3 = ($$anchor$4) => {
							var span_2 = root_7();
							var text_3 = child(span_2, true);
							reset(span_2);
							template_effect(() => set_text(text_3, value()));
							append($$anchor$4, span_2);
						};
						var alternate_1 = ($$anchor$4) => {
							var fragment_4 = comment();
							var node_5 = first_child(fragment_4);
							var consequent_4 = ($$anchor$5) => {
								var div_2 = root_9();
								var button = child(div_2);
								button.__click = (e) => {
									e.stopPropagation();
									handleEdit(record());
								};
								var button_1 = sibling(button, 2);
								button_1.__click = (e) => {
									e.stopPropagation();
									handleDelete(record());
								};
								reset(div_2);
								append($$anchor$5, div_2);
							};
							var alternate = ($$anchor$5) => {
								var text_4 = text();
								template_effect(() => set_text(text_4, value()));
								append($$anchor$5, text_4);
							};
							if_block(node_5, ($$render) => {
								if (col().id === "actions") $$render(consequent_4);
								else $$render(alternate, false);
							}, true);
							append($$anchor$4, fragment_4);
						};
						if_block(node_4, ($$render) => {
							if (record()._updated && col().header === "Apellidos") $$render(consequent_3);
							else $$render(alternate_1, false);
						}, true);
						append($$anchor$3, fragment_3);
					};
					if_block(node_3, ($$render) => {
						if (col().header === "Edad" && record().edad > 50) $$render(consequent_2);
						else $$render(alternate_2, false);
					});
					append($$anchor$2, fragment_2);
				};
				VTable(node_2, {
					get columns() {
						return columns;
					},
					get data() {
						return get(data);
					},
					maxHeight: "calc(100vh - 16rem)",
					onRowClick: handleRowClick,
					get selected() {
						return get(selectedRecord);
					},
					isSelected: isRowSelected,
					estimateSize: 34,
					overscan: 15,
					cellRenderer,
					$$slots: { cellRenderer: true }
				});
			}
			template_effect(($0) => set_text(text$1, `Testing with ${$0 ?? ""} rows - Svelte 5 + Custom Virtualizer`), [() => get(data).length.toLocaleString()]);
			append($$anchor$1, fragment_1);
		},
		$$slots: { default: true }
	});
	pop();
}
delegate(["click"]);
export { _page as component };
