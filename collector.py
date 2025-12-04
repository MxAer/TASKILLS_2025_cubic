import os
import sys

# Настройки игнорирования
IGNORE_DIRS = {
    'node_modules', '.git', 'venv', '__pycache__', '.idea', '.vscode', 'dist', 'build', 'tmp', 'vendor'
}
IGNORE_FILES = {
    'package-lock.json', 'yarn.lock', 'go.sum', '.DS_Store', 'collector.py', 'code_bundle.txt'
}
IGNORE_EXTENSIONS = {
    '.png', '.jpg', '.jpeg', '.gif', '.ico', '.svg', '.eot', '.ttf', '.woff', '.woff2', '.exe', '.bin', '.dll'
}

OUTPUT_FILE = 'code_bundle.txt'

def should_ignore(path, name, is_dir):
    if is_dir:
        return name in IGNORE_DIRS
    
    if name in IGNORE_FILES:
        return True
    
    _, ext = os.path.splitext(name)
    if ext.lower() in IGNORE_EXTENSIONS:
        return True
        
    return False

def generate_tree(start_path, prefix=""):
    tree_str = ""
    try:
        entries = os.listdir(start_path)
    except PermissionError:
        return ""
    
    # Сортируем: папки сверху, потом файлы
    entries = sorted(entries, key=lambda e: (not os.path.isdir(os.path.join(start_path, e)), e.lower()))
    
    filtered_entries = [e for e in entries if not should_ignore(start_path, e, os.path.isdir(os.path.join(start_path, e)))]
    
    for i, entry in enumerate(filtered_entries):
        path = os.path.join(start_path, entry)
        is_last = (i == len(filtered_entries) - 1)
        
        connector = "└── " if is_last else "├── "
        tree_str += f"{prefix}{connector}{entry}\n"
        
        if os.path.isdir(path):
            extension = "    " if is_last else "│   "
            tree_str += generate_tree(path, prefix + extension)
            
    return tree_str

def collect_files(start_path):
    content = ""
    for root, dirs, files in os.walk(start_path):
        # Фильтрация папок in-place
        dirs[:] = [d for d in dirs if d not in IGNORE_DIRS]
        
        for file in files:
            if should_ignore(root, file, False):
                continue
                
            file_path = os.path.join(root, file)
            relative_path = os.path.relpath(file_path, start_path)
            
            try:
                with open(file_path, 'r', encoding='utf-8') as f:
                    file_content = f.read()
                    
                content += "\n" + "="*50 + "\n"
                content += f"FILE: {relative_path}\n"
                content += "="*50 + "\n"
                content += file_content + "\n"
            except Exception as e:
                print(f"Skipping binary or unreadable file: {file_path}")
                
    return content

def main():
    root_dir = os.getcwd()
    print(f"Collecting project from: {root_dir}")
    
    # 1. Generate Tree
    tree_visualization = f"Project Structure ({os.path.basename(root_dir)}):\n"
    tree_visualization += "." + "\n"
    tree_visualization += generate_tree(root_dir)
    
    # 2. Collect Content
    file_contents = collect_files(root_dir)
    
    # 3. Write Output
    full_output = tree_visualization + "\n\n" + file_contents
    
    with open(OUTPUT_FILE, 'w', encoding='utf-8') as f:
        f.write(full_output)
        
    print(f"Done! Code bundle saved to: {OUTPUT_FILE}")

if __name__ == "__main__":
    main()
