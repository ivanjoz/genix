import os
import glob

def main():
    base_path = 'libs/index_builder'
    productos_file = os.path.join(base_path, 'productos.psv')
    chunks_dir = os.path.join(base_path, 'parsed_chunks')
    output_file = os.path.join(base_path, 'productos_con_precios.psv')

    # Load prices from chunks
    prices = {}
    chunk_files = glob.glob(os.path.join(chunks_dir, '*.psv'))
    
    print(f"Reading {len(chunk_files)} chunk files from {chunks_dir}...")
    for chunk_file in chunk_files:
        with open(chunk_file, 'r', encoding='utf-8') as f:
            for line in f:
                parts = line.strip().split('|')
                if len(parts) >= 4:
                    product = parts[0].strip()
                    try:
                        price = float(parts[3].strip())
                        prices[product] = price
                    except ValueError:
                        continue # Skip invalid lines

    print(f"Loaded {len(prices)} unique product prices.")

    # Process main file
    print(f"Processing {productos_file}...")
    matched_count = 0
    missing_products = []
    
    with open(productos_file, 'r', encoding='utf-8') as fin, \
         open(output_file, 'w', encoding='utf-8') as fout:
        
        # Read header
        header = fin.readline().strip()
        
        # Write new header
        fout.write("product|brand|categories|price\n")
        
        for line in fin:
            if not line.strip():
                continue
                
            parts = line.strip().split('|')
            if len(parts) >= 3:
                product = parts[0].strip()
                
                if product in prices:
                    new_price = prices[product] * 3
                    # Format matching user request: product | brand | categories | price * 3
                    fout.write(f"{product}|{parts[1]}|{parts[2]}|{new_price:.2f}\n")
                    matched_count += 1
                else:
                    missing_products.append(product)

    print(f"Finished. Matched {matched_count} records.")
    print(f"Output written to {output_file}")
    
    if missing_products:
        print(f"\nFound {len(missing_products)} products without prices:")
        for product in missing_products:
            print(f"MISSING PRICE: {product}")

if __name__ == "__main__":
    main()
