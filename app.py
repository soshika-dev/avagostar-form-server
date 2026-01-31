from internal.app import create_app
from internal.config import Config


if __name__ == "__main__":
    config = Config()
    host, port = "0.0.0.0", 8080
    if ":" in config.http_addr:
        _, port_str = config.http_addr.split(":", 1)
        if port_str:
            port = int(port_str)
    app = create_app()
    app.run(host=host, port=port, debug=config.env != "prod")
