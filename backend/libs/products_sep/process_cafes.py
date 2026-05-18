import csv

def categorize_cafes():
    input_file = 'cafes_e_infusiones.psv'
    
    capsulas_keywords = ['cápsulas', 'capsulas', 'cápsula', 'capsula', 'nespresso', 'dolce gusto', 'tassimo', 't-disc']
    molido_grano_keywords = ['molido', 'en grano', 'natural', 'mezcla', 'tostado']
    infusiones_tes_keywords = ['infusión', 'infusion', 'té', 'te', 'poleo', 'manzanilla', 'menta', 'tila', 'rooibos', 'hierbas']
    cacao_solubles_keywords = ['cacao', 'soluble', 'cola cao', 'nesquik', 'instantáneo', 'instantaneo', 'batido']

    capsulas = []
    molido_grano = []
    infusiones_tes = []
    cacao_solubles = []
    otros = []

    with open(input_file, 'r', encoding='utf-8') as f:
        reader = csv.DictReader(f, delimiter='|')
        for row in reader:
            name = row['Producto'].lower()
            categories = row['Categories']
            
            if any(k in name for k in capsulas_keywords):
                row['Categories'] = f"{categories}, Café en Cápsulas"
                capsulas.append(row)
            elif any(k in name for k in cacao_solubles_keywords):
                row['Categories'] = f"{categories}, Cacao y Solubles"
                cacao_solubles.append(row)
            elif any(k in name for k in infusiones_tes_keywords):
                row['Categories'] = f"{categories}, Infusiones y Té"
                infusiones_tes.append(row)
            elif any(k in name for k in molido_grano_keywords):
                row['Categories'] = f"{categories}, Café Molido y en Grano"
                molido_grano.append(row)
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

    write_psv('cafes_capsulas_procesed.psv', capsulas)
    write_psv('cafes_molido_y_grano_procesed.psv', molido_grano)
    write_psv('cafes_infusiones_y_te_procesed.psv', infusiones_tes)
    write_psv('cafes_cacao_y_solubles_procesed.psv', cacao_solubles)
    write_psv('cafes_otros_procesed.psv', otros)

    print(f"Cápsulas: {len(capsulas)}")
    print(f"Molido/Grano: {len(molido_grano)}")
    print(f"Infusiones/Té: {len(infusiones_tes)}")
    print(f"Cacao/Solubles: {len(cacao_solubles)}")
    print(f"Otros: {len(otros)}")

if __name__ == "__main__":
    categorize_cafes()
