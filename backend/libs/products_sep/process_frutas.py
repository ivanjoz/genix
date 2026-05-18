import csv

def categorize_frutas_verduras():
    input_file = 'frutas_y_verduras.psv'
    
    frutas_keywords = [
        'fruta', 'arándanos', 'arandanos', 'banana', 'limón', 'limon', 'manzana', 
        'naranja', 'pera', 'plátano', 'platano', 'uva', 'fresa', 'kiwi', 'mango',
        'piña', 'pina', 'melón', 'melon', 'sandía', 'sandia', 'nectarina', 
        'melocotón', 'melocoton', 'mandarina', 'cereza', 'ciruela', 'higo'
    ]
    verduras_keywords = [
        'verdura', 'acelga', 'ajo', 'alcachofa', 'apio', 'berenjena', 'bimi', 
        'brócoli', 'brocoli', 'calabacín', 'calabacin', 'calabaza', 'canónigos',
        'canonigos', 'cebolla', 'champiñón', 'champinon', 'col', 'coliflor', 
        'espárrago', 'esparrago', 'espinaca', 'judía', 'judia', 'lechuga', 
        'patata', 'pepino', 'pimiento', 'puerro', 'tomate', 'zanahoria', 'boniato'
    ]
    preparados_keywords = ['ensalada', 'preparado', 'mezcla', 'sopa', 'gazpacho', 'salmorejo', 'guacamole', 'hummus']
    # Everything else (setas, algas, hierbas aromáticas, etc.) goes to Otros

    frutas = []
    verduras = []
    preparados = []
    otros = []

    with open(input_file, 'r', encoding='utf-8') as f:
        reader = csv.DictReader(f, delimiter='|')
        for row in reader:
            name = row['Producto'].lower()
            categories = row['Categories']
            
            if any(k in name for k in preparados_keywords):
                row['Categories'] = f"{categories}, Ensaladas y Preparados"
                preparados.append(row)
            elif any(k in name for k in frutas_keywords):
                row['Categories'] = f"{categories}, Frutas"
                frutas.append(row)
            elif any(k in name for k in verduras_keywords):
                row['Categories'] = f"{categories}, Verduras y Hortalizas"
                verduras.append(row)
            else:
                row['Categories'] = f"{categories}, Otros"
                otros.append(row)

    def write_psv(filename, data):
        if not data: return
        keys = ['Producto', 'Brand', 'Categories']
        with open(filename, 'w', encoding='utf-8', newline='') as f:
            writer = csv.DictWriter(f, fieldnames=keys, delimiter='|')
            writer.writeheader()
            writer.writerows(data)

    write_psv('frutas_y_verduras_frutas_procesed.psv', frutas)
    write_psv('frutas_y_verduras_verduras_procesed.psv', verduras)
    write_psv('frutas_y_verduras_preparados_procesed.psv', preparados)
    write_psv('frutas_y_verduras_otros_procesed.psv', otros)

    print(f"Frutas: {len(frutas)}")
    print(f"Verduras: {len(verduras)}")
    print(f"Preparados: {len(preparados)}")
    print(f"Otros: {len(otros)}")

if __name__ == "__main__":
    categorize_frutas_verduras()
