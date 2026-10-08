import csv
import os


def load_csv_data(file_path):
    data = []
    if not os.path.exists(file_path):
        raise FileNotFoundError(f"Archivo de datos no encontrado: {file_path}")

    with open(file_path, mode='r', encoding='utf-8') as f:
        reader = csv.DictReader(f)
        for row in reader:
            data.append(row)
    return data
