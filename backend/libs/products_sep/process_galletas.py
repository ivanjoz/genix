import csv

def categorize_galletas():
    input_file = 'galletas_y_cereales.psv'
    
    galletas_keywords = ['galleta', 'galletas', 'barquillos', 'cookies', 'dinosaurus', 'marbú', 'chips ahoy', 'oreo']
    cereales_keywords = ['cereales', 'avena', 'muesli', 'corn flakes', 'chocopic', 'weetabix', 'kellogg', 'muesly', 'special k']
    barritas_keywords = ['barrita', 'barritas']
    # Everything else (tortitas, etc.) goes to Tortitas y Otros

    galletas = []
    cereales = []
    barritas = []
    tortitas_otros = []

    with open(input_file, 'r', encoding='utf-8') as f:
        reader = csv.DictReader(f, delimiter='|')
        for row in reader:
            name = row['Producto'].lower()
            categories = row['Categories']
            
            if any(k in name for k in barritas_keywords):
                row['Categories'] = f"{categories}, Barritas"
                barritas.append(row)
            elif any(k in name for k in galletas_keywords):
                row['Categories'] = f"{categories}, Galletas"
                galletas.append(row)
            elif any(k in name for k in cereales_keywords):
                row['Categories'] = f"{categories}, Cereales"
                cereales.append(row)
            else:
                row['Categories'] = f"{categories}, Tortitas y Otros"
                tortitas_otros.append(row)

    def write_psv(filename, data):
        if not data: return
        keys = ['Producto', 'Brand', 'Categories']
        with open(filename, 'w', encoding='utf-8', newline='') as f:
            writer = csv.DictWriter(f, fieldnames=keys, delimiter='|')
            writer.writeheader()
            writer.writerows(data)

    write_psv('galletas_y_cereales_galletas_procesed.psv', galletas)
    write_psv('galletas_y_cereales_cereales_procesed.psv', cereales)
    write_psv('galletas_y_cereales_barritas_procesed.psv', barritas)
    write_psv('galletas_y_cereales_tortitas_y_otros_procesed.psv', tortitas_otros)

    print(f"Galletas: {len(galletas)}")
    print(f"Cereales: {len(cereales)}")
    print(f"Barritas: {len(barritas)}")
    print(f"Tortitas y Otros: {len(tortitas_otros)}")

if __name__ == "__main__":
    categorize_galletas()
