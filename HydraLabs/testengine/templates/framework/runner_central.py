import argparse
import subprocess
import os
import sys


def main():
    parser = argparse.ArgumentParser(description="Orquestador Maestro de Automatización DOM")
    parser.add_argument(
        '--suite',
        choices=['__APP__', 'all'],
        default='__APP__',
        help='Suite a ejecutar.'
    )
    parser.add_argument(
        '--headless',
        action='store_true',
        help='Ejecutar el navegador en modo sin interfaz.'
    )
    args = parser.parse_args()

    root_dir = os.path.dirname(os.path.abspath(__file__))
    reports_dir = os.path.join(root_dir, "reports")
    os.makedirs(reports_dir, exist_ok=True)

    tests_dir = os.path.join(root_dir, "tests")
    app_dir = os.path.join(root_dir, "apps", "__APP__")

    test_targets = []
    report_name = "report_unificado.html"

    if args.suite == '__APP__':
        test_targets.append(os.path.join(tests_dir, "__APP__"))
        report_name = "report___APP__.html"
    elif args.suite == 'all':
        test_targets.append(tests_dir)
        report_name = "report_unificado.html"

    report_path = os.path.join(reports_dir, report_name)

    if test_targets:
        print(f"\n==============================================")
        print(f">> INICIANDO EJECUCIÓN UNIFICADA: {args.suite.upper()} <<")
        print(f"==============================================")

        env = os.environ.copy()
        env["PYTHONPATH"] = f"{root_dir}{os.pathsep}{app_dir}"
        env["ENV_FILE"] = os.path.join(root_dir, ".env")

        if args.headless:
            env["HEADLESS"] = "true"

        cmd = [
            sys.executable, "-m", "pytest",
            *test_targets,
            f"--html={report_path}",
            "--self-contained-html",
            "-v"
        ]

        subprocess.run(cmd, env=env, cwd=root_dir)
    else:
        print("No se especificaron objetivos válidos para la prueba.")


if __name__ == "__main__":
    main()
