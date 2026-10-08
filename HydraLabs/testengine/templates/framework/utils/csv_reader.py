import csv
import os


def cargar_datos_csv(file_path):
    datos = []
    if not os.path.exists(file_path):
        print(f"[!] Error: El archivo {file_path} no existe.")
        return datos

    with open(file_path, mode='r', encoding='utf-8') as f:
        reader = csv.reader(f)
        try:
            header = next(reader)
        except StopIteration:
            return datos

        for row in reader:
            if row:
                procesada = list(row)
                for i, valor in enumerate(procesada):
                    if isinstance(valor, str) and valor.startswith("ENV:"):
                        env_key = valor.split("ENV:")[1]
                        procesada[i] = os.getenv(env_key, valor)

                datos.append(tuple(procesada))
    return datos
