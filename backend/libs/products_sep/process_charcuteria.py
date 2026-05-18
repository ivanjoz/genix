import csv
import io

def categorize_charcuteria():
    input_file = 'charcuteria.psv'
    
    embutidos_keywords = [
        'chorizo', 'fuet', 'salchichón', 'salchichon', 'chistorra', 'longaniza', 
        'butifarra', 'sobrasada', 'morcilla', 'lomo', 'cecina', 'espetec', 
        'secallona', 'caña de lomo', 'cabecero de lomo', 'salchicha', 'salami',
        'pepperoni', 'compango', 'embutido', 'surtido'
    ]
    
    cocidos_fiambres_keywords = [
        'cocido', 'fiambre', 'pavo', 'pollo', 'mortadela', 'chopped', 
        'pechuga', 'cabeza de jabalí', 'cabeza de jabali', 'cabeza de cerdo',
        'chicharrón', 'chicharron', 'galantina', 'lacón', 'lacon', 'oreja',
        'york'
    ]
    
    pates_foies_keywords = [
        'paté', 'pate', 'foie', 'foei', 'mousse', 'bloc', 'hígado', 'higado', 'untable'
    ]
    
    bacon_keywords = [
        'bacon', 'bacón', 'sajonia', 'corteza', 'torrezno', 'panceta'
    ]

    jamon_paleta_keywords = [
        'jamón', 'jamon', 'paleta'
    ]

    embutidos = []
    cocidos_fiambres = []
    pates_foies = []
    bacon_otros = []
    jamon_paleta = []
    remaining = []

    with open(input_file, 'r', encoding='utf-8') as f:
        reader = csv.DictReader(f, delimiter='|')
        for row in reader:
            name = row['Producto'].lower()
            categories = row['Categories']
            
            if any(k in name for k in pates_foies_keywords):
                row['Categories'] = f"{categories}, Patés y Foies"
                pates_foies.append(row)
            elif any(k in name for k in bacon_keywords):
                row['Categories'] = f"{categories}, Bacon y Otros"
                bacon_otros.append(row)
            elif any(k in name for k in cocidos_fiambres_keywords):
                row['Categories'] = f"{categories}, Cocidos y Fiambres"
                cocidos_fiambres.append(row)
            elif any(k in name for k in embutidos_keywords):
                row['Categories'] = f"{categories}, Embutidos"
                embutidos.append(row)
            elif any(k in name for k in jamon_paleta_keywords):
                row['Categories'] = f"{categories}, Jamón y Paleta"
                jamon_paleta.append(row)
            elif 'vegalia' in name or 'palitos de mar' in name or 'huevo hilado' in name:
                row['Categories'] = f"{categories}, Otros"
                remaining.append(row)
            else:
                remaining.append(row)

    def write_psv(filename, data):
        if not data: return
        keys = ['Producto', 'Brand', 'Categories']
        with open(filename, 'w', encoding='utf-8', newline='') as f:
            writer = csv.DictWriter(f, fieldnames=keys, delimiter='|')
            writer.writeheader()
            writer.writerows(data)

    write_psv('charcuteria_embutidos_procesed.psv', embutidos)
    write_psv('charcuteria_cocidos_y_fiambres_procesed.psv', cocidos_fiambres)
    write_psv('charcuteria_pates_y_foies_procesed.psv', pates_foies)
    write_psv('charcuteria_bacon_y_otros_procesed.psv', bacon_otros)
    write_psv('charcuteria_jamon_y_paleta_rest_procesed.psv', jamon_paleta)
    write_psv('charcuteria_otros_restantes.psv', remaining)

    print(f"Embutidos: {len(embutidos)}")
    print(f"Cocidos y Fiambres: {len(cocidos_fiambres)}")
    print(f"Patés y Foies: {len(pates_foies)}")
    print(f"Bacon y Otros: {len(bacon_otros)}")
    print(f"Jamón y Paleta (restantes): {len(jamon_paleta)}")
    print(f"Restantes: {len(remaining)}")

if __name__ == "__main__":
    categorize_charcuteria()
