# This file defines the shared configuration for tools and system instructions
# to ensure consistency between training data generation, training, and inference.

SYSTEM_INSTRUCTION = (
    "El modelo es un asistente para ejecutar acciones del sistema Genix en español. "
    "Es un ERP para pequeñas empresas que permite gestión de inventarios, ventas, compras, productos, cajas y finanzas. "
    "Si no entiendes la acción que el usuario desea realizar, responde exactamente con: no entendí la acción que deseas realizar. "
    "You are a model that can do function calling with the following functions"
)

ALL_TOOLS = [
    {
        "type": "function",
        "function": {
            "name": "get_producto_stock",
            "description": "Obtener el stock de un producto en todos los almacenes",
            "parameters": {
                "type": "object",
                "properties": {
                    "producto_nombre": {"type": "string", "description": "Nombre del producto"}
                },
                "required": ["producto_nombre"]
            }
        }
    },
    {
        "type": "function",
        "function": {
            "name": "get_producto_movimientos",
            "description": "Obtener los últimos movimientos de inventario de un producto",
            "parameters": {
                "type": "object",
                "properties": {
                    "producto_nombre": {"type": "string", "description": "Nombre del producto"}
                },
                "required": ["producto_nombre"]
            }
        }
    },
    {
        "type": "function",
        "function": {
            "name": "do_agregar_producto",
            "description": "Redirigir al formulario para agregar un nuevo producto",
            "parameters": {
                "type": "object",
                "properties": {
                    "producto_nombre": {"type": "string", "description": "Nombre del producto"}
                }
            }
        }
    },
    {
        "type": "function",
        "function": {
            "name": "get_caja_saldo",
            "description": "Obtener el saldo actual de una caja específica",
            "parameters": {
                "type": "object",
                "properties": {
                    "caja_nombre": {"type": "string", "description": "Nombre de la caja"}
                },
                "required": ["caja_nombre"]
            }
        }
    },
    {
        "type": "function",
        "function": {
            "name": "get_caja_movimientos",
            "description": "Obtener los últimos movimientos de una caja específica",
            "parameters": {
                "type": "object",
                "properties": {
                    "caja_nombre": {"type": "string", "description": "Nombre de la caja"}
                },
                "required": ["caja_nombre"]
            }
        }
    },
    {
        "type": "function",
        "function": {
            "name": "get_ventas_del_dia",
            "description": "Obtener un resumen de las ventas realizadas el día de hoy",
            "parameters": {
                "type": "object",
                "properties": {}
            }
        }
    },
    {
        "type": "function",
        "function": {
            "name": "get_ventas_ultimos_dias",
            "description": "Obtener un resumen de las ventas de los últimos días",
            "parameters": {
                "type": "object",
                "properties": {
                    "num_dias": {"type": "integer", "description": "Número de días a consultar"}
                }
            }
        }
    },
    {
        "type": "function",
        "function": {
            "name": "do_agregar_movimiento_caja",
            "description": "Abrir formulario para registrar un movimiento de caja (ingreso/egreso)",
            "parameters": {
                "type": "object",
                "properties": {
                    "caja_nombre": {"type": "string", "description": "Nombre de la caja"},
                    "monto": {"type": "number", "description": "Monto del movimiento"}
                }
            }
        }
    },
    {
        "type": "function",
        "function": {
            "name": "do_agregar_gasto",
            "description": "Abrir formulario para registrar un nuevo gasto",
            "parameters": {
                "type": "object",
                "properties": {
                    "descripcion": {"type": "string", "description": "Descripción del gasto"},
                    "monto": {"type": "number", "description": "Monto del gasto"}
                },
                "required": ["descripcion"]
            }
        }
    }
]

