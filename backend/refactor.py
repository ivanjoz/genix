import os
import re
import glob
from collections import defaultdict

def main():
    # 1. Map structs to their modules
    struct_to_module = {}
    
    # Scrape struct definitions from backend/*/types/*.go
    types_files = glob.glob('backend/*/types/*.go')
    for tf in types_files:
        module = tf.split('/')[1]  # backend/<module>/types
        with open(tf, 'r') as f:
            content = f.read()
            # find all `type X` where X is capitalized
            matches = re.findall(r'^type\s+([A-Z]\w*)', content, re.MULTILINE)
            for m in matches:
                struct_to_module[m] = module

    # add some known aliases
    struct_to_module['AlmacenStockMin'] = 'negocio'

    files = glob.glob('backend/**/*.go', recursive=True)
    
    for file in files:
        if 'types/' in file and file.endswith('.go'):
            continue # skip types directories
            
        with open(file, 'r') as f:
            content = f.read()
            
        original_content = content
        
        # Simple replacements
        content = content.replace('"app/operaciones"', '"app/logistica"')
        
        # remove "app/handlers" except in exec/
        if 'exec/' not in file:
            content = re.sub(r'^\s*"app/handlers"\n', '', content, flags=re.MULTILINE)
            
        # Detect aliases for "app/types"
        # it can be `s "app/types"` or `"app/types"` or `types "app/types"`
        types_import_match = re.search(r'([a-zA-Z0-9_]*)\s*"app/types"', content)
        if not types_import_match:
            # Maybe the file was modified, but let's check if it needs saving
            if content != original_content:
                with open(file, 'w') as f:
                    f.write(content)
            continue
            
        alias = types_import_match.group(1).strip()
        if alias == '':
            alias = 'types' # default package name if no alias
            
        # find all `alias.XXX` usages
        pattern = rf'\b{alias}\.([A-Z]\w*)\b'
        used_structs = set(re.findall(pattern, content))
        
        modules_needed = set()
        for s in used_structs:
            if s in struct_to_module:
                modules_needed.add(struct_to_module[s])
            else:
                print(f"Warning: struct {s} in {file} not found in any types module!")
                # assume it's in negocio as fallback?
                modules_needed.add('negocio')
                struct_to_module[s] = 'negocio'

        # Generate new imports
        new_imports = []
        for mod in sorted(list(modules_needed)):
            new_imports.append(f'{mod}Types "app/{mod}/types"')
            
        new_imports_str = '\n\t'.join(new_imports)
        
        # Replace the import "app/types" line with new_imports_str
        content = re.sub(r'([a-zA-Z0-9_]*)\s*"app/types"', new_imports_str, content, count=1)
        
        # Replace `alias.XXX` with `moduleTypes.XXX`
        for s in used_structs:
            mod = struct_to_module[s]
            content = re.sub(rf'\b{alias}\.{s}\b', f'{mod}Types.{s}', content)

        if content != original_content:
            with open(file, 'w') as f:
                f.write(content)

if __name__ == '__main__':
    main()
