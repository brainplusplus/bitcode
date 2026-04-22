import sys
import json
import importlib.util
import traceback


def load_script(script_path):
    spec = importlib.util.spec_from_file_location("plugin_script", script_path)
    if spec is None:
        return None
    mod = importlib.util.module_from_spec(spec)
    spec.loader.exec_module(mod)
    return mod


def execute_script(script_path, params):
    try:
        mod = load_script(script_path)
        if mod is None:
            return {"warning": "script not found", "script": script_path}

        if hasattr(mod, "execute"):
            return mod.execute(params)
        elif hasattr(mod, "main"):
            return mod.main(params)
        else:
            return {"executed": True, "script": script_path}
    except Exception as e:
        raise e


def main():
    sys.stderr.write("[plugin:python] ready\n")
    sys.stderr.flush()

    for line in sys.stdin:
        line = line.strip()
        if not line:
            continue

        try:
            request = json.loads(line)
        except json.JSONDecodeError:
            response = {"error": "invalid JSON", "id": 0}
            sys.stdout.write(json.dumps(response) + "\n")
            sys.stdout.flush()
            continue

        req_id = request.get("id", 0)
        method = request.get("method", "")
        params = request.get("params", {})

        if method == "execute":
            try:
                script_path = params.get("script", "")
                script_params = params.get("params", {})
                result = execute_script(script_path, script_params)
                response = {"result": result, "id": req_id}
            except Exception as e:
                response = {"error": str(e), "id": req_id}
        else:
            response = {"error": f"unknown method: {method}", "id": req_id}

        sys.stdout.write(json.dumps(response) + "\n")
        sys.stdout.flush()


if __name__ == "__main__":
    main()
