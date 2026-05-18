import csv

def categorize_bebidas():
    input_file = 'bebidas_refrescantes_y_gaseosas.psv'
    
    cola_gaseosas_keywords = [
        'cola', 'coca-cola', 'pepsi', 'gaseosa', '7up', 'sprite', 'fanta', 'naranja', 
        'limón', 'limon', 'kas', 'schweppes', 'refresco', 'sifon', 'sifón'
    ]
    isotonicas_energeticas_keywords = [
        'aquarius', 'powerade', 'gatorade', 'monster', 'red bull', 'burn', 
        'energética', 'energetica', 'isotónica', 'isotonica', 'isodrink'
    ]
    tes_aguas_aloe_keywords = [
        'té', 'te', 'nestea', 'lipton', 'aloe', 'coco', 'agua con gas', 
        'agua mineral con', 'solán', 'solan', 'horchata', 'chufa'
    ]
    # Everything else (bitter, tónicas, etc.) goes to Tónicas y Otros

    cola_gaseosas = []
    isotonicas_energeticas = []
    tes_aguas_aloe = []
    tonicas_otros = []

    with open(input_file, 'r', encoding='utf-8') as f:
        reader = csv.DictReader(f, delimiter='|')
        for row in reader:
            name = row['Producto'].lower()
            categories = row['Categories']
            
            if any(k in name for k in isotonicas_energeticas_keywords):
                row['Categories'] = f"{categories}, Isotónicas y Energéticas"
                isotonicas_energeticas.append(row)
            elif any(k in name for k in tes_aguas_aloe_keywords):
                row['Categories'] = f"{categories}, Tés, Aguas y Aloe"
                tes_aguas_aloe.append(row)
            elif any(k in name for k in cola_gaseosas_keywords):
                row['Categories'] = f"{categories}, Cola y Gaseosas"
                cola_gaseosas.append(row)
            else:
                row['Categories'] = f"{categories}, Tónicas y Otros"
                tonicas_otros.append(row)

    def write_psv(filename, data):
        if not data: return
        keys = ['Producto', 'Brand', 'Categories']
        with open(filename, 'w', encoding='utf-8', newline='') as f:
            writer = csv.DictWriter(f, fieldnames=keys, delimiter='|')
            writer.writeheader()
            writer.writerows(data)

    write_psv('bebidas_cola_y_gaseosas_procesed.psv', cola_gaseosas)
    write_psv('bebidas_isotonicas_y_energeticas_procesed.psv', isotonicas_energeticas)
    write_psv('bebidas_tes_aguas_y_aloe_procesed.psv', tes_aguas_aloe)
    write_psv('bebidas_tonicas_y_otros_procesed.psv', tonicas_otros)

    print(f"Cola y Gaseosas: {len(cola_gaseosas)}")
    print(f"Isotónicas y Energéticas: {len(isotonicas_energeticas)}")
    print(f"Tés, Aguas y Aloe: {len(tes_aguas_aloe)}")
    print(f"Tónicas y Otros: {len(tonicas_otros)}")

if __name__ == "__main__":
    categorize_bebidas()
