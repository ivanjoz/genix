training_data = [
    # 1. get_producto_stock
    {
        "developer": "You are a model that can do function calling with the following functions<start_function_declaration>call:get_producto_stock{description:<escape>Obtener el stock de un producto en todos los almacenes<escape>,parameters:{properties:{producto_nombre:{type:<escape>STRING<escape>}},required:[<escape>producto_nombre<escape>]}}<end_function_declaration>",
        "assistant": "<start_function_call>call:get_producto_stock{{producto_nombre:<escape>[VARIABLE_1]<escape>}}<end_function_call>",
        "queries": [
            {"user": "¿Cuál es el stock del producto Arroz Faraón de 5kg?", "variables": ["Arroz Faraón de 5kg"]},
            {"user": "Dime cuánto queda de Inka Kola 3 litros en el inventario", "variables": ["Inka Kola 3 litros"]},
            {"user": "Stock de Leche Gloria azul pack de 6", "variables": ["Leche Gloria azul pack de 6"]},
            {"user": "Consulta la cantidad disponible de Aceite Primor Premium", "variables": ["Aceite Primor Premium"]},
            {"user": "¿Cuántas unidades tenemos de Detergente Opal 1kg?", "variables": ["Detergente Opal 1kg"]},
            {"user": "ver stock de cemento sol", "variables": ["cemento sol"]},
            {"user": "¿Cuántos sacos de Azúcar rubia quedan?", "variables": ["Azúcar rubia"]},
            {"user": "Busca el stock de Coca Cola 500ml", "variables": ["Coca Cola 500ml"]},
            {"user": "Consultar existencia de Galletas Oreo", "variables": ["Galletas Oreo"]},
            {"user": "¿Qué cantidad hay de Papas Lay's familiares?", "variables": ["Papas Lay's familiares"]},
            {"user": "Saldo en almacén de Atún Primor", "variables": ["Atún Primor"]},
            {"user": "Dime el stock actual de Cerveza Cristal", "variables": ["Cerveza Cristal"]}
        ]
    },
    # 2. get_producto_movimientos
    {
        "developer": "You are a model that can do function calling with the following functions<start_function_declaration>call:get_producto_movimientos{description:<escape>Obtener los últimos movimientos de inventario de un producto<escape>,parameters:{properties:{producto_nombre:{type:<escape>STRING<escape>}},required:[<escape>producto_nombre<escape>]}}<end_function_declaration>",
        "assistant": "<start_function_call>call:get_producto_movimientos{{producto_nombre:<escape>[VARIABLE_1]<escape>}}<end_function_call>",
        "queries": [
            {"user": "¿Cuáles son los movimientos de inventario del producto Clavos de 2 pulgadas?", "variables": ["Clavos de 2 pulgadas"]},
            {"user": "Muéstrame el historial de entradas y salidas de Atún Real", "variables": ["Atún Real"]},
            {"user": "Ver kárdex de Cerveza Pilsen Callao caja", "variables": ["Cerveza Pilsen Callao caja"]},
            {"user": "¿Qué pasó con el stock de Galletas Soda Field?", "variables": ["Galletas Soda Field"]},
            {"user": "últimos movimientos de pintura Vencedor blanco", "variables": ["pintura Vencedor blanco"]},
            {"user": "Ver registros de entradas del producto Fierro de 1/2", "variables": ["Fierro de 1/2"]},
            {"user": "¿Qué salidas hubo de Arroz Costeño?", "variables": ["Arroz Costeño"]},
            {"user": "Reporte de kárdex de Leche Ideal amanecer", "variables": ["Leche Ideal amanecer"]},
            {"user": "Quiero ver los movimientos de Yogurt Danlac fresa", "variables": ["Yogurt Danlac fresa"]},
            {"user": "¿Hubo salidas de Detergente Ariel hoy?", "variables": ["Detergente Ariel"]},
            {"user": "Ver histórico de stock de Panetón Gloria", "variables": ["Panetón Gloria"]},
            {"user": "Consulta movimientos de Aceite Cil", "variables": ["Aceite Cil"]}
        ]
    },
    # 3. do_agregar_producto
    {
        "developer": "You are a model that can do function calling with the following functions<start_function_declaration>call:do_agregar_producto{description:<escape>Redirigir al formulario para agregar un nuevo producto<escape>,parameters:{properties:{producto_nombre:{type:<escape>STRING<escape>}}}}<end_function_declaration>",
        "assistant": "<start_function_call>call:do_agregar_producto{{producto_nombre:<escape>[VARIABLE_1]<escape>}}<end_function_call>",
        "queries": [
            {"user": "Agrega el producto Chocolate Sublime", "variables": ["Chocolate Sublime"]},
            {"user": "Quiero registrar un nuevo producto: Yogurt Gloria fresa", "variables": ["Yogurt Gloria fresa"]},
            {"user": "Añadir producto nuevo", "variables": [""]},
            {"user": "Crea una ficha para Papel Higiénico Elite 40 unidades", "variables": ["Papel Higiénico Elite 40 unidades"]},
            {"user": "Pon en el sistema el Paneton D'Onofrio", "variables": ["Paneton D'Onofrio"]},
            {"user": "Registrar un artículo nuevo", "variables": [""]},
            {"user": "Crear producto Galletas Casino", "variables": ["Galletas Casino"]},
            {"user": "Añade al catálogo el Whisky Red Label", "variables": ["Whisky Red Label"]},
            {"user": "Agrega el producto nuevo: Gaseosa Pepsi 2L", "variables": ["Gaseosa Pepsi 2L"]},
            {"user": "Quiero meter un producto nuevo al sistema", "variables": [""]},
            {"user": "Regístrame este producto: Jabón Bolívar", "variables": ["Jabón Bolívar"]},
            {"user": "Nuevo producto: Papel Toalla Elite", "variables": ["Papel Toalla Elite"]}
        ]
    },
    # 4. get_caja_saldo
    {
        "developer": "You are a model that can do function calling with the following functions<start_function_declaration>call:get_caja_saldo{description:<escape>Obtener el saldo actual de una caja específica<escape>,parameters:{properties:{caja_nombre:{type:<escape>STRING<escape>}},required:[<escape>caja_nombre<escape>]}}<end_function_declaration>",
        "assistant": "<start_function_call>call:get_caja_saldo{{caja_nombre:<escape>[VARIABLE_1]<escape>}}<end_function_call>",
        "queries": [
            {"user": "¿Cuánto dinero hay en la Caja Principal?", "variables": ["Caja Principal"]},
            {"user": "Saldo actual de la Caja Mostrador 1", "variables": ["Caja Mostrador 1"]},
            {"user": "¿Qué monto tenemos en la Caja de Ventas?", "variables": ["Caja de Ventas"]},
            {"user": "Ver efectivo en Caja Chica", "variables": ["Caja Chica"]},
            {"user": "cuanto hay en caja 2", "variables": ["caja 2"]},
            {"user": "Saldo en Caja 1", "variables": ["Caja 1"]},
            {"user": "¿Cuánto hay en la Caja de Efectivo?", "variables": ["Caja de Efectivo"]},
            {"user": "Dime el total de la Caja de Turno Mañana", "variables": ["Caja de Turno Mañana"]},
            {"user": "Ver saldo de Caja Soles", "variables": ["Caja Soles"]},
            {"user": "Consulta la plata que hay en Caja Lima", "variables": ["Caja Lima"]},
            {"user": "¿Cuál es el balance de la Caja Administrativa?", "variables": ["Caja Administrativa"]},
            {"user": "Saldo de Caja General", "variables": ["Caja General"]}
        ]
    },
    # 5. get_caja_movimientos
    {
        "developer": "You are a model that can do function calling with the following functions<start_function_declaration>call:get_caja_movimientos{description:<escape>Obtener los últimos movimientos de una caja específica<escape>,parameters:{properties:{caja_nombre:{type:<escape>STRING<escape>}},required:[<escape>caja_nombre<escape>]}}<end_function_declaration>",
        "assistant": "<start_function_call>call:get_caja_movimientos{{caja_nombre:<escape>[VARIABLE_1]<escape>}}<end_function_call>",
        "queries": [
            {"user": "¿Cuáles fueron los movimientos de la Caja Principal?", "variables": ["Caja Principal"]},
            {"user": "Ver historial de transacciones de la Caja Mostrador", "variables": ["Caja Mostrador"]},
            {"user": "Dime qué pagos y cobros hubo en la Caja de Ventas hoy", "variables": ["Caja de Ventas"]},
            {"user": "muéstrame el reporte de la caja central", "variables": ["caja central"]},
            {"user": "Movimientos de la Caja 2", "variables": ["Caja 2"]},
            {"user": "¿Qué transacciones se hicieron en la Caja de Ventas ayer?", "variables": ["Caja de Ventas"]},
            {"user": "Ver kárdex de la Caja Principal", "variables": ["Caja Principal"]},
            {"user": "Reporte de ingresos de la Caja Chica", "variables": ["Caja Chica"]},
            {"user": "Movimientos de hoy en Caja Sucursal", "variables": ["Caja Sucursal"]},
            {"user": "Historial de la Caja Administrativa", "variables": ["Caja Administrativa"]},
            {"user": "¿Qué entradas hubo en la Caja General?", "variables": ["Caja General"]},
            {"user": "Muéstrame los pagos de Caja 1", "variables": ["Caja 1"]}
        ]
    },
    # 6. get_ventas_del_dia
    {
        "developer": "You are a model that can do function calling with the following functions<start_function_declaration>call:get_ventas_del_dia{description:<escape>Obtener un resumen de las ventas realizadas el día de hoy<escape>,parameters:{properties:{},required:[]}}<end_function_declaration>",
        "assistant": "<start_function_call>call:get_ventas_del_dia{}<end_function_call>",
        "queries": [
            {"user": "¿Cuáles fueron las ventas del día de hoy?", "variables": []},
            {"user": "Dame el resumen de ventas de hoy", "variables": []},
            {"user": "¿Cuánto hemos vendido hoy?", "variables": []},
            {"user": "Ver ventas diarias", "variables": []},
            {"user": "reporte de ventas de este momento", "variables": []},
            {"user": "¿Cómo van las ventas hoy?", "variables": []},
            {"user": "Resumen de lo vendido este día", "variables": []},
            {"user": "Dime el total de ventas de hoy", "variables": []},
            {"user": "Reporte diario de ventas", "variables": []},
            {"user": "¿Cuánto dinero ha ingresado por ventas hoy?", "variables": []},
            {"user": "Ver lo que se ha vendido en el día", "variables": []},
            {"user": "Totales de ventas del día", "variables": []}
        ]
    },
    # 7. get_ventas_ultimos_dias
    {
        "developer": "You are a model that can do function calling with the following functions<start_function_declaration>call:get_ventas_ultimos_dias{description:<escape>Obtener un resumen de las ventas de los últimos días<escape>,parameters:{properties:{num_dias:{type:<escape>INTEGER<escape>}}}}<end_function_declaration>",
        "assistant": "<start_function_call>call:get_ventas_ultimos_dias{{num_dias:[VARIABLE_1]}}<end_function_call>",
        "queries": [
            {"user": "¿Cuáles fueron las ventas de los últimos 8 días?", "variables": [8]},
            {"user": "Resumen de ventas de la última semana", "variables": [7]},
            {"user": "Dime cuánto se vendió en los últimos 3 días", "variables": [3]},
            {"user": "Ventas de los últimos 15 días", "variables": [15]},
            {"user": "ver ventas de 30 dias", "variables": [30]},
            {"user": "Ventas de los últimos 2 días", "variables": [2]},
            {"user": "¿Cuánto se vendió en los pasados 5 días?", "variables": [5]},
            {"user": "Reporte de ventas de los últimos 10 días", "variables": [10]},
            {"user": "Dame las ventas de la quincena", "variables": [15]},
            {"user": "¿Qué tal las ventas de los últimos 4 días?", "variables": [4]},
            {"user": "Ventas de la semana pasada", "variables": [7]},
            {"user": "Ver ventas de los últimos 20 días", "variables": [20]}
        ]
    },
    # 8. do_agregar_movimiento_caja
    {
        "developer": "You are a model that can do function calling with the following functions<start_function_declaration>call:do_agregar_movimiento_caja{description:<escape>Abrir formulario para registrar un movimiento de caja (ingreso/egreso)<escape>,parameters:{properties:{caja_nombre:{type:<escape>STRING<escape>},monto:{type:<escape>NUMBER<escape>}}}}<end_function_declaration>",
        "assistant": "<start_function_call>call:do_agregar_movimiento_caja{{caja_nombre:<escape>[VARIABLE_1]<escape>,monto:[VARIABLE_2]}}<end_function_call>",
        "queries": [
            {"user": "Agrega un pago de 150 soles a la Caja mostrador", "variables": ["Caja mostrador", 150]},
            {"user": "Agrega un cobro de 180 soles a la Caja principal", "variables": ["Caja principal", 180]},
            {"user": "Recibí 100 soles, agrégalo a la Caja principal", "variables": ["Caja principal", 100]},
            {"user": "Registra un ingreso de 50 soles en Caja Chica", "variables": ["Caja Chica", 50]},
            {"user": "Salida de 20 soles de la Caja Mostrador 2", "variables": ["Caja Mostrador 2", 20]},
            {"user": "Cobro de 200 soles en Caja Principal", "variables": ["Caja Principal", 200]},
            {"user": "Ingreso de 300 soles a la Caja de Ventas", "variables": ["Caja de Ventas", 300]},
            {"user": "Saca 50 soles de la Caja Chica", "variables": ["Caja Chica", 50]},
            {"user": "Registra un pago recibido de 120 en Caja 1", "variables": ["Caja 1", 120]},
            {"user": "Agrega salida de 45 soles a Caja Mostrador", "variables": ["Caja Mostrador", 45]},
            {"user": "Cobré 80 soles, ponlo en Caja Chica", "variables": ["Caja Chica", 80]},
            {"user": "Pagaron 500 soles a la Caja Administrativa", "variables": ["Caja Administrativa", 500]}
        ]
    },
    # 9. do_agregar_gasto
    {
        "developer": "You are a model that can do function calling with the following functions<start_function_declaration>call:do_agregar_gasto{description:<escape>Abrir formulario para registrar un nuevo gasto<escape>,parameters:{properties:{descripcion:{type:<escape>STRING<escape>},monto:{type:<escape>NUMBER<escape>}},required:[<escape>descripcion<escape>]}}<end_function_declaration>",
        "assistant": "<start_function_call>call:do_agregar_gasto{{descripcion:<escape>[VARIABLE_1]<escape>,monto:[VARIABLE_2]}}<end_function_call>",
        "queries": [
            {"user": "Agrega un gasto por Cena de navidad de 400 soles", "variables": ["Cena de navidad", 400]},
            {"user": "Registrar gasto de 50 soles por limpieza", "variables": ["limpieza", 50]},
            {"user": "Gasto en pasajes de 15 soles", "variables": ["pasajes", 15]},
            {"user": "Pagué el recibo de luz de 250 soles, anótalo como gasto", "variables": ["recibo de luz", 250]},
            {"user": "Nuevo gasto por compra de útiles de oficina", "variables": ["compra de útiles de oficina", "null"]},
            {"user": "Gasto de 80 soles por movilidad", "variables": ["movilidad", 80]},
            {"user": "Registra un gasto: Pago de internet 120 soles", "variables": ["Pago de internet", 120]},
            {"user": "Añade gasto por compra de café", "variables": ["compra de café", "null"]},
            {"user": "Gasto de 300 soles por mantenimiento", "variables": ["mantenimiento", 300]},
            {"user": "Pago de arbitrios 450 soles", "variables": ["Pago de arbitrios", 450]},
            {"user": "Registrar gasto por útiles de aseo 40 soles", "variables": ["útiles de aseo", 40]},
            {"user": "Anotar gasto: Pago de luz 200 soles", "variables": ["Pago de luz", 200]}
        ]
    }
]
