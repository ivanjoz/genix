import csv

def categorize_higiene():
    input_file = 'higiene_y_bano.psv'
    
    geles_jabones_keywords = ['gel', 'jabón', 'jabon', 'ducha', 'baño', 'bano', 'mousse', 'sales de baño']
    desodorantes_keywords = ['desodorante', 'roll on', 'spray']
    papel_panuelos_keywords = ['papel', 'pañuelo', 'pañuelos', 'higiénico', 'higienico', 'servilleta']
    # Everything else (esponjas, bastoncillos, cepillos, tijeras, etc.) goes to Accesorios y Otros

    geles_jabones = []
    desodorantes = []
    papel_panuelos = []
    accesorios_otros = []

    with open(input_file, 'r', encoding='utf-8') as f:
        reader = csv.DictReader(f, delimiter='|')
        for row in reader:
            name = row['Producto'].lower()
            categories = row['Categories']
            
            if any(k in name for k in desodorantes_keywords):
                row['Categories'] = f"{categories}, Desodorantes"
                desodorantes.append(row)
            elif any(k in name for k in papel_panuelos_keywords):
                row['Categories'] = f"{categories}, Papel y Pañuelos"
                papel_panuelos.append(row)
            elif any(k in name for k in geles_jabones_keywords):
                row['Categories'] = f"{categories}, Geles y Jabones"
                geles_jabones.append(row)
            else:
                row['Categories'] = f"{categories}, Accesorios y Otros"
                accesorios_otros.append(row)

    def write_psv(filename, data):
        if not data: return
        keys = ['Producto', 'Brand', 'Categories']
        with open(filename, 'w', encoding='utf-8', newline='') as f:
            writer = csv.DictWriter(f, fieldnames=keys, delimiter='|')
            writer.writeheader()
            writer.writerows(data)

    write_psv('higiene_geles_y_jabones_procesed.psv', geles_jabones)
    write_psv('higiene_desodorantes_procesed.psv', desodorantes)
    write_psv('higiene_papel_y_panuelos_procesed.psv', papel_panuelos)
    write_psv('higiene_accesorios_y_otros_procesed.psv', accesorios_otros)

    print(f"Geles y Jabones: {len(geles_jabones)}")
    print(f"Desodorantes: {len(desodorantes)}")
    print(f"Papel y Pañuelos: {len(papel_panuelos)}")
    print(f"Accesorios y Otros: {len(accesorios_otros)}")

if __name__ == "__main__":
    categorize_higiene()
