import csv

def categorize_destilados():
    input_file = 'bebidas_destiladas.psv'
    
    whisky_ron_ginebra_keywords = ['whisky', 'ron', 'ginebra', 'gin', 'j&b', 'ballantine', 'beefeater', 'bacardi', 'havana']
    licores_cremas_keywords = ['licor', 'crema', 'baileys', 'orujo', 'hierbas', 'limoncello', 'pacharán', 'pacharan', 'amaretto']
    brandy_conac_anis_keywords = ['brandy', 'coñac', 'conac', 'anís', 'anis', 'chinchón', 'chinchon', 'cognac']
    # Everything else (vodka, tequila, aperitivos, cócteles, accesorios) goes to Otros

    whisky_ron_ginebra = []
    licores_cremas = []
    brandy_conac_anis = []
    otros = []

    with open(input_file, 'r', encoding='utf-8') as f:
        reader = csv.DictReader(f, delimiter='|')
        for row in reader:
            name = row['Producto'].lower()
            categories = row['Categories']
            
            if any(k in name for k in whisky_ron_ginebra_keywords):
                row['Categories'] = f"{categories}, Whisky, Ron y Ginebra"
                whisky_ron_ginebra.append(row)
            elif any(k in name for k in licores_cremas_keywords):
                row['Categories'] = f"{categories}, Licores y Cremas"
                licores_cremas.append(row)
            elif any(k in name for k in brandy_conac_anis_keywords):
                row['Categories'] = f"{categories}, Brandy, Coñac y Anís"
                brandy_conac_anis.append(row)
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

    write_psv('destilados_whisky_ron_ginebra_procesed.psv', whisky_ron_ginebra)
    write_psv('destilados_licores_y_cremas_procesed.psv', licores_cremas)
    write_psv('destilados_brandy_conac_y_anis_procesed.psv', brandy_conac_anis)
    write_psv('destilados_otros_procesed.psv', otros)

    print(f"Whisky/Ron/Ginebra: {len(whisky_ron_ginebra)}")
    print(f"Licores/Cremas: {len(licores_cremas)}")
    print(f"Brandy/Coñac/Anís: {len(brandy_conac_anis)}")
    print(f"Otros: {len(otros)}")

if __name__ == "__main__":
    categorize_destilados()
