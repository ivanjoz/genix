import csv

def categorize_cabello():
    input_file = 'cuidado_del_cabello.psv'
    
    champu_keywords = ['champú', 'champu']
    acondicionador_mascarilla_keywords = ['acondicionador', 'mascarilla', 'bálsamo', 'balsamo', 'ampollas', 'serum', 'sérum', 'tratamiento', 'reparador', 'nutritivo']
    coloracion_keywords = ['tinte', 'coloración', 'coloracion', 'aclarante', 'retoca', 'mechas']
    # Everything else will go to "Fijación, Accesorios y Otros"

    champu = []
    acondicionador_mascarilla = []
    coloracion = []
    otros = []

    with open(input_file, 'r', encoding='utf-8') as f:
        reader = csv.DictReader(f, delimiter='|')
        for row in reader:
            name = row['Producto'].lower()
            categories = row['Categories']
            
            if any(k in name for k in champu_keywords):
                row['Categories'] = f"{categories}, Champú"
                champu.append(row)
            elif any(k in name for k in acondicionador_mascarilla_keywords):
                row['Categories'] = f"{categories}, Acondicionador y Mascarilla"
                acondicionador_mascarilla.append(row)
            elif any(k in name for k in coloracion_keywords):
                row['Categories'] = f"{categories}, Coloración"
                coloracion.append(row)
            else:
                row['Categories'] = f"{categories}, Fijación, Accesorios y Otros"
                otros.append(row)

    def write_psv(filename, data):
        if not data: return
        keys = ['Producto', 'Brand', 'Categories']
        with open(filename, 'w', encoding='utf-8', newline='') as f:
            writer = csv.DictWriter(f, fieldnames=keys, delimiter='|')
            writer.writeheader()
            writer.writerows(data)

    write_psv('cuidado_del_cabello_champu_procesed.psv', champu)
    write_psv('cuidado_del_cabello_acondicionador_y_mascarilla_procesed.psv', acondicionador_mascarilla)
    write_psv('cuidado_del_cabello_coloracion_procesed.psv', coloracion)
    write_psv('cuidado_del_cabello_fijacion_y_accesorios_procesed.psv', otros)

    print(f"Champú: {len(champu)}")
    print(f"Acondicionador/Mascarilla: {len(acondicionador_mascarilla)}")
    print(f"Coloración: {len(coloracion)}")
    print(f"Fijación/Accesorios/Otros: {len(otros)}")

if __name__ == "__main__":
    categorize_cabello()
