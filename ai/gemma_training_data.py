# This file defines the training data configurations.
# It is used by gemma_generator.py to create the final JSONL file.

training_data = [
    # 1. get_producto_stock
    {
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
            {"user": "Dime el stock actual de Cerveza Cristal", "variables": ["Cerveza Cristal"]},
            {"user": "¿Tenemos Fideos Don Vittorio?", "variables": ["Fideos Don Vittorio"]},
            {"user": "Quanto hay de Papel Higiénico Suave", "variables": ["Papel Higiénico Suave"]},
            {"user": "stock de Ron Cartavio", "variables": ["Ron Cartavio"]}
        ]
    },
    # 2. get_producto_movimientos
    {
        "assistant": "<start_function_call>call:get_producto_movimientos{{producto_nombre:<escape>[VARIABLE_1]<escape>}}<end_function_call>",
        "queries": [
            {"user": "¿Cuáles son los movimientos de inventario del producto Clavos de 2 pulgadas?", "variables": ["Clavos de 2 pulgadas"]},
            {"user": "Muéstrame el historial de entradas y salidas de Atún Real", "variables": ["Atún Real"]},
            {"user": "Ver kárdex de Cerveza Pilsen Callao caja", "variables": ["Cerveza Pilsen Callao caja"]},
            {"user": "últimos movimientos de pintura Vencedor blanco", "variables": ["pintura Vencedor blanco"]},
            {"user": "Ver registros de entradas del producto Fierro de 1/2", "variables": ["Fierro de 1/2"]},
            {"user": "Historial de Azúcar Blanca", "variables": ["Azúcar Blanca"]},
            {"user": "¿Qué entradas tuvo el Café Nescafé?", "variables": ["Café Nescafé"]}
        ]
    },
    # 3. do_agregar_producto
    {
        "assistant": "<start_function_call>call:do_agregar_producto{{producto_nombre:<escape>[VARIABLE_1]<escape>}}<end_function_call>",
        "queries": [
            {"user": "Agrega el producto Chocolate Sublime", "variables": ["Chocolate Sublime"]},
            {"user": "Quiero registrar un nuevo producto: Yogurt Gloria fresa", "variables": ["Yogurt Gloria fresa"]},
            {"user": "Añadir producto nuevo", "variables": [""]},
            {"user": "Crea una ficha para Papel Higiénico Elite 40 unidades", "variables": ["Papel Higiénico Elite 40 unidades"]},
            {"user": "Pon en el sistema el Paneton D'Onofrio", "variables": ["Paneton D'Onofrio"]},
            {"user": "Registrar un artículo nuevo", "variables": [""]},
            {"user": "Crear producto Galletas Casino", "variables": ["Galletas Casino"]}
        ]
    },
    # 4. get_caja_saldo
    {
        "assistant": "<start_function_call>call:get_caja_saldo{{caja_nombre:<escape>[VARIABLE_1]<escape>}}<end_function_call>",
        "queries": [
            {"user": "¿Cuánto dinero hay en la Caja Principal?", "variables": ["Caja Principal"]},
            {"user": "Saldo actual de la Caja Mostrador 1", "variables": ["Caja Mostrador 1"]},
            {"user": "¿Qué monto tenemos en la Caja de Ventas?", "variables": ["Caja de Ventas"]},
            {"user": "Ver efectivo en Caja Chica", "variables": ["Caja Chica"]},
            {"user": "cuanto hay en caja 2", "variables": ["caja 2"]},
            {"user": "Balance de Caja Sucursal Norte", "variables": ["Caja Sucursal Norte"]}
        ]
    },
    # 5. get_caja_movimientos
    {
        "assistant": "<start_function_call>call:get_caja_movimientos{{caja_nombre:<escape>[VARIABLE_1]<escape>}}<end_function_call>",
        "queries": [
            {"user": "¿Cuáles fueron los movimientos de la Caja Principal?", "variables": ["Caja Principal"]},
            {"user": "Ver historial de transacciones de la Caja Mostrador", "variables": ["Caja Mostrador"]},
            {"user": "Dime qué pagos y cobros hubo en la Caja de Ventas hoy", "variables": ["Caja de Ventas"]},
            {"user": "muéstrame el reporte de la caja central", "variables": ["caja central"]},
            {"user": "Movimientos de la Caja 2", "variables": ["Caja 2"]}
        ]
    },
    # 6. get_ventas_del_dia
    {
        "assistant": "<start_function_call>call:get_ventas_del_dia{}<end_function_call>",
        "queries": [
            {"user": "¿Cuáles fueron las ventas del día de hoy?", "variables": []},
            {"user": "Dame el resumen de ventas de hoy", "variables": []},
            {"user": "¿Cuánto hemos vendido hoy?", "variables": []},
            {"user": "Ver ventas diarias", "variables": []},
            {"user": "reporte de ventas de este momento", "variables": []}
        ]
    },
    # 7. get_ventas_ultimos_dias
    {
        "assistant": "<start_function_call>call:get_ventas_ultimos_dias{{num_dias:[VARIABLE_1]}}<end_function_call>",
        "queries": [
            {"user": "¿Cuáles fueron las ventas de los últimos 8 días?", "variables": [8]},
            {"user": "Resumen de ventas de la última semana", "variables": [7]},
            {"user": "Dime cuánto se vendió en los últimos 3 días", "variables": [3]},
            {"user": "Ventas de los últimos 15 días", "variables": [15]},
            {"user": "ver ventas de 30 dias", "variables": [30]},
            {"user": "Ventas de los últimos 2 días", "variables": [2]},
            {"user": "Reporte semanal", "variables": [7]},
            {"user": "Dame un resumen de 7 días", "variables": [7]},
            # Added more explicit "days" markers to avoid confusion with "soles"
            {"user": "Ver reporte de ventas de los pasados 120 días", "variables": [120]},
            {"user": "Consulta ventas del último periodo de 90 días", "variables": [90]},
            {"user": "¿Cuánto vendimos en un lapso de 60 días?", "variables": [60]}
        ]
    },
    # 8. do_agregar_movimiento_caja
    {
        "assistant": "<start_function_call>call:do_agregar_movimiento_caja{{caja_nombre:<escape>[VARIABLE_1]<escape>,monto:[VARIABLE_2]}}<end_function_call>",
        "queries": [
            {"user": "Agrega un pago de 150 soles a la Caja mostrador", "variables": ["Caja mostrador", 150]},
            {"user": "Agrega un cobro de 180 soles a la Caja principal", "variables": ["Caja principal", 180]},
            {"user": "Recibí 100 soles, agrégalo a la Caja principal", "variables": ["Caja principal", 100]},
            {"user": "Registra un ingreso de 50 soles en Caja Chica", "variables": ["Caja Chica", 50]},
            {"user": "Salida de 20 soles de la Caja Mostrador 2", "variables": ["Caja Mostrador 2", 20]},
            {"user": "Registra un pago recibido de 120 en Caja 1", "variables": ["Caja 1", 120]},
            {"user": "Me dieron 110 soles, ponlos en Caja 3", "variables": ["Caja 3", 110]}
        ]
    },
    # 9. do_agregar_gasto
    {
        "assistant": "<start_function_call>call:do_agregar_gasto{{descripcion:<escape>[VARIABLE_1]<escape>,monto:[VARIABLE_2]}}<end_function_call>",
        "queries": [
            {"user": "Agrega un gasto por Cena de navidad de 400 soles", "variables": ["Cena de navidad", 400]},
            {"user": "Registrar gasto de 50 soles por limpieza", "variables": ["limpieza", 50]},
            {"user": "Gasto en pasajes de 15 soles", "variables": ["pasajes", 15]},
            {"user": "Pagué el recibo de luz de 250 soles, anótalo como gasto", "variables": ["recibo de luz", 250]},
            {"user": "Gasto de 80 soles por movilidad", "variables": ["movilidad", 80]},
            {"user": "Registra un gasto: Pago de internet 120 soles", "variables": ["Pago de internet", 120]},
            {"user": "Pago de arbitrios 450 soles", "variables": ["Pago de arbitrios", 450]},
            {"user": "Registrar 120 soles de almuerzo navideño", "variables": ["almuerzo navideño", 120]},
            {"user": "Gasté 65 soles en combustible", "variables": ["combustible", 65]},
            {"user": "Pagué 350 de alquiler, regístralo", "variables": ["alquiler", 350]}
        ]
    },
    # 10. Casos no entendidos (Fallback)
    {
        "assistant": "no entendí la acción que deseas realizar",
        "queries": [
            {"user": "Hola, ¿cómo estás?", "variables": []},
            {"user": "¿Cuál es el sentido de la vida?", "variables": []},
            {"user": "Cuéntame un chiste", "variables": []},
            {"user": "¿Qué hora es?", "variables": []},
            {"user": "Dime el precio del dólar hoy", "variables": []},
            {"user": "Adiós, nos vemos", "variables": []}
        ]
    }
]
