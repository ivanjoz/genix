import { $ as proxy, A as if_block, L as delegate, M as append, P as from_html, Q as sibling, Tt as __toESM, W as template_effect, X as child, Z as first_child, _t as reset, at as state, gt as next, j as set_text, lt as pop, rt as set, st as user_derived, ut as push, z as get } from "../chunks/CTB4HzdN.js";
import "../chunks/D25ese0K.js";
import "../chunks/2yAUi6s0.js";
import { b as formatN, h as POST, m as GetHandler, v as sendServiceMessage, y as Env } from "../chunks/BwrZ3UQO.js";
import { o as require_notiflix_aio_3_2_8_min } from "../chunks/DwKmqXYU.js";
import { a as formatTime, t as ConfirmWarn } from "../chunks/DOFkf9MZ.js";
import { t as Page } from "../chunks/GeoukCMt.js";
import { t as VTable } from "../chunks/BCJF-hdW.js";
import "../chunks/DnSz4jvr.js";
var import_notiflix_aio_3_2_8_min = /* @__PURE__ */ __toESM(require_notiflix_aio_3_2_8_min());
var BackupsService = class extends GetHandler {
	route = "backups";
	useCache = {
		min: 0,
		ver: 1
	};
	keyID = "Name";
	#backups = state(proxy([]));
	get backups() {
		return get(this.#backups);
	}
	set backups(value) {
		set(this.#backups, value, true);
	}
	handler(records) {
		records.sort((a, b) => a.Name > b.Name ? 1 : -1);
		this.backups = records || [];
	}
	constructor() {
		super();
		this.fetch();
	}
	refreshBackups() {
		this.fetch();
	}
};
const createBackup = () => {
	return POST({
		data: {},
		route: "backup-create",
		successMessage: "Backup generado exitosamente",
		errorMessage: "Error al generar el backup"
	});
};
const restoreBackup = (name) => {
	return POST({
		data: { Name: name },
		route: "backup-restore",
		successMessage: "Backup restaurado exitosamente",
		errorMessage: "Error al restaurar el backup"
	});
};
var root_2 = from_html(`<div class="w-full c-red ff-bold flex-wrap box-error-ms mt-16">Seleccione un Backup</div>`);
var root_3 = from_html(`<div class="flex w-full justify-between"><div><div class="w-full flex-wrap ff-semibold"> </div> <div class="text-gray-600 mt-4"> </div></div> <button class="bx-purple" aria-label="Descargar backup"><i class="icon-download"></i></button></div> <div class="flex justify-center w-full mt-16"><button class="bx-blue">Restaurar <i class="icon-database"></i></button></div>`, 1);
var root_1 = from_html(`<div class="w-full grid gap-20" style="grid-template-columns: 4fr 3fr;"><div><div class="flex items-center justify-between mb-6"><div class="h2 ff-bold">Backups</div> <div class="flex items-center"><button class="bx-green mr-8" aria-label="Generar backup"><i class="icon-plus"></i></button> <button class="bx-blue" aria-label="Subir backup"><i class="icon-upload"></i></button></div></div> <!></div> <div style="margin-left: 20px;"><div class="mb-6 h2 ff-bold">Restore</div> <div class="_1 w-full rounded-[8px] min-h-160 py-12 px-16 aMe"><!></div></div></div>`);
function _page($$anchor, $$props) {
	push($$props, true);
	const backupsService = new BackupsService();
	let backupSelected = state(null);
	const generarBackup = async () => {
		import_notiflix_aio_3_2_8_min.Loading.standard("Generando Backup...");
		try {
			await createBackup();
			backupsService.refreshBackups();
		} catch (error) {}
		import_notiflix_aio_3_2_8_min.Loading.remove();
	};
	const restaurar = async (name) => {
		import_notiflix_aio_3_2_8_min.Loading.standard("Restaurando Backup...");
		try {
			await restoreBackup(name);
			await sendServiceMessage(26, {});
		} catch (error) {}
		import_notiflix_aio_3_2_8_min.Loading.remove();
	};
	const downloadBackup = (backup) => {
		const s3key = [
			"backups",
			1,
			backup.Name
		].join("/");
		const url = Env.S3_URL + s3key;
		console.log("url to download::", url);
		const aElement = document.createElement("a");
		aElement.setAttribute("download", backup.Name);
		aElement.href = url;
		aElement.setAttribute("target", "_blank");
		aElement.click();
		aElement.remove();
	};
	const columns = [
		{
			header: "Created",
			headerCss: "w-176",
			cellCss: "px-6 nowrap",
			getValue: (e) => formatTime(e.upd, "Y-m-d h:n")
		},
		{
			header: "Nombre",
			highlight: true,
			cellCss: "px-6",
			getValue: (e) => e.Name
		},
		{
			header: "Tamaño",
			headerCss: "w-120",
			cellCss: "text-center",
			getValue: (e) => `${formatN(e.Size / 1e3 / 1e3, 2)} mb`
		}
	];
	Page($$anchor, {
		title: "Backups & Restore",
		children: ($$anchor$1, $$slotProps) => {
			var div = root_1();
			var div_1 = child(div);
			var div_2 = child(div_1);
			var div_3 = sibling(child(div_2), 2);
			var button = child(div_3);
			button.__click = (ev) => {
				ev.stopPropagation();
				ConfirmWarn("Generar Backup", "¿Desea generar el backup ahora?", "SI", "NO", () => {
					generarBackup();
				});
			};
			next(2);
			reset(div_3);
			reset(div_2);
			var node = sibling(div_2, 2);
			{
				let $0 = user_derived(() => get(backupSelected) || void 0);
				VTable(node, {
					get data() {
						return backupsService.backups;
					},
					get columns() {
						return columns;
					},
					css: "selectable w-full",
					tableCss: "cursor-pointer",
					maxHeight: "calc(80vh - 13rem)",
					get selected() {
						return get($0);
					},
					isSelected: (e, selected) => {
						if (!selected || typeof selected === "number") return false;
						return e.Name === selected.Name;
					},
					onRowClick: (row) => {
						if (get(backupSelected)?.Name === row.Name) set(backupSelected, null);
						else set(backupSelected, row, true);
					}
				});
			}
			reset(div_1);
			var div_4 = sibling(div_1, 2);
			var div_5 = sibling(child(div_4), 2);
			var node_1 = child(div_5);
			var consequent = ($$anchor$2) => {
				append($$anchor$2, root_2());
			};
			var alternate = ($$anchor$2) => {
				var fragment_1 = root_3();
				var div_7 = first_child(fragment_1);
				var div_8 = child(div_7);
				var div_9 = child(div_8);
				var text = child(div_9, true);
				reset(div_9);
				var div_10 = sibling(div_9, 2);
				var text_1 = child(div_10);
				reset(div_10);
				reset(div_8);
				var button_1 = sibling(div_8, 2);
				button_1.__click = (ev) => {
					ev.stopPropagation();
					downloadBackup(get(backupSelected));
				};
				reset(div_7);
				var div_11 = sibling(div_7, 2);
				var button_2 = child(div_11);
				button_2.__click = (ev) => {
					ev.stopPropagation();
					ConfirmWarn("Restaurar Backup", `Restaurar el backup realizado el ${formatTime(get(backupSelected).upd, "Y-m-d h:n")}`, "SI", "NO", () => {
						restaurar(get(backupSelected).Name);
					});
				};
				reset(div_11);
				template_effect(($0) => {
					set_text(text, get(backupSelected).Name);
					set_text(text_1, `${$0 ?? ""} mb`);
				}, [() => formatN(get(backupSelected).Size / 1e3 / 1e3, 2)]);
				append($$anchor$2, fragment_1);
			};
			if_block(node_1, ($$render) => {
				if (!get(backupSelected)) $$render(consequent);
				else $$render(alternate, false);
			});
			reset(div_5);
			reset(div_4);
			reset(div);
			append($$anchor$1, div);
		},
		$$slots: { default: true }
	});
	pop();
}
delegate(["click"]);
export { _page as component };
