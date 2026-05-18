import csv

def categorize_pescaderia():
    input_file = 'pescaderia.psv'
    
    pescado_keywords = [
        'atún', 'atun', 'bacalao', 'boquerones', 'boqueron', 'merluza', 'salmón', 
        'salmon', 'dorada', 'lubina', 'trucha', 'sardina', 'emperador', 'pez espada',
        'lenguado', 'pescadilla', 'bacaladilla', 'bonito', 'rape'
    ]
    marisco_molusco_keywords = [
        'mejillones', 'mejillon', 'almeja', 'langostino', 'gamba', 'cigala', 
        'calamar', 'chipirón', 'chipiron', 'pota', 'potón', 'poton', 'pulpo', 
        'sepia', 'cangrejo', 'buey', 'vieira', 'berberecho', 'navaja', 'langosta', 
        'percebe', 'bogavante'
    ]
    preparados_ahumados_keywords = [
        'ahumado', 'preparado', 'surimi', 'gula', 'anguriña', 'angurina', 
        'brandada', 'albóndigas', 'albondigas', 'croquetas', 'vinagre', 
        'salpicón', 'salpicon', 'ceviche', 'sushi'
    ]
    congelados_keywords = ['congelado', 'congelados', 'ultracongelado', 'congelada']

    pescado = []
    marisco_molusco = []
    preparados_ahumados = []
    congelados = []
    otros = []

    with open(input_file, 'r', encoding='utf-8') as f:
        reader = csv.DictReader(f, delimiter='|')
        for row in reader:
            name = row['Producto'].lower()
            categories = row['Categories']
            
            if any(k in name for k in preparados_ahumados_keywords):
                row['Categories'] = f"{categories}, Preparados y Ahumados"
                preparados_ahumados.append(row)
            elif any(k in name for k in congelados_keywords):
                row['Categories'] = f"{categories}, Congelados"
                congelados.append(row)
            elif any(k in name for k in pescado_keywords):
                row['Categories'] = f"{categories}, Pescado"
                pescado.append(row)
            elif any(k in name for k in marisco_molusco_keywords):
                row['Categories'] = f"{categories}, Marisco y Moluscos"
                marisco_molusco.append(row)
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

    write_psv('pescaderia_pescado_procesed.psv', pescado)
    write_psv('pescaderia_marisco_y_moluscos_procesed.psv', marisco_molusco)
    write_psv('pescaderia_preparados_y_ahumados_procesed.psv', preparados_ahumados)
    write_psv('pescaderia_congelados_procesed.psv', congelados)
    write_psv('pescaderia_otros_procesed.psv', otros)

    print(f"Pescado: {len(pescado)}")
    print(f"Marisco/Moluscos: {len(marisco_molusco)}")
    print(f"Preparados/Ahumados: {len(preparados_ahumados)}")
    print(f"Congelados: {len(congelados)}")
    print(f"Otros: {len(otros)}")

if __name__ == "__main__":
    categorize_pescaderia()
