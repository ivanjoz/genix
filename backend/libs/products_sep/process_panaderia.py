import csv

def categorize_panaderia():
    input_file = 'panaderia_y_bolleria.psv'
    
    pan_tostadas_keywords = [
        'pan', 'baguette', 'barra', 'hogaza', 'pulguitas', 'viena', 'chapata',
        'pistola', 'biscotte', 'tostada', 'rebanada', 'picos', 'colines', 'biscotes',
        'regañás', 'regañas', 'mold'
    ]
    bolleria_dulce_keywords = [
        'berlina', 'donut', 'ensaimada', 'croissant', 'napolitana', 'sobao',
        'magdalena', 'bizcocho', 'palmera', 'caña', 'donuts', 'muffin',
        'donettes', 'pastelito', 'bollycao', 'pantera rosa', 'tigretón',
        'tigreton', 'brioche'
    ]
    pasteleria_tartas_keywords = [
        'tarta', 'pastel', 'hojaldre', 'crema', 'chocolate', 'postre', 
        'rosquillas', 'pestiños', 'rosquille', 'mantecado', 'polvorón',
        'polvoron', 'turrón', 'turron', 'mazapán', 'mazapan'
    ]
    # Everything else (salado, arepas, empanadas, etc.) goes to Salado y Otros

    pan_tostadas = []
    bolleria_dulce = []
    pasteleria_tartas = []
    otros = []

    with open(input_file, 'r', encoding='utf-8') as f:
        reader = csv.DictReader(f, delimiter='|')
        for row in reader:
            name = row['Producto'].lower()
            categories = row['Categories']
            
            # Prioritize Pan if it's basic bread
            if any(k in name for k in pan_tostadas_keywords) and not any(k in name for k in pasteleria_tartas_keywords):
                row['Categories'] = f"{categories}, Pan y Tostadas"
                pan_tostadas.append(row)
            elif any(k in name for k in bolleria_dulce_keywords):
                row['Categories'] = f"{categories}, Bollería Dulce"
                bolleria_dulce.append(row)
            elif any(k in name for k in pasteleria_tartas_keywords):
                row['Categories'] = f"{categories}, Pastelería y Tartas"
                pasteleria_tartas.append(row)
            else:
                row['Categories'] = f"{categories}, Salado y Otros"
                otros.append(row)

    def write_psv(filename, data):
        if not data: return
        keys = ['Producto', 'Brand', 'Categories']
        with open(filename, 'w', encoding='utf-8', newline='') as f:
            writer = csv.DictWriter(f, fieldnames=keys, delimiter='|')
            writer.writeheader()
            writer.writerows(data)

    write_psv('panaderia_pan_y_tostadas_procesed.psv', pan_tostadas)
    write_psv('panaderia_bolleria_dulce_procesed.psv', bolleria_dulce)
    write_psv('panaderia_pasteleria_y_tartas_procesed.psv', pasteleria_tartas)
    write_psv('panaderia_salado_y_otros_procesed.psv', otros)

    print(f"Pan y Tostadas: {len(pan_tostadas)}")
    print(f"Bollería Dulce: {len(bolleria_dulce)}")
    print(f"Pastelería y Tartas: {len(pasteleria_tartas)}")
    print(f"Salado y Otros: {len(otros)}")

if __name__ == "__main__":
    categorize_panaderia()
