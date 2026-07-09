from mlx_lm import load,batch_generate
from http.server import BaseHTTPRequestHandler,HTTPServer
import json

MODEL_NAME = "mlx-community/Qwen2.5-0.5B-Instruct-4bit"

print(f"Loading model: {MODEL_NAME}...")
model, tokenizer = load(MODEL_NAME)
print("Model loaded.")

class BatchHandler(BaseHTTPRequestHandler):
    def do_POST(self):
        content_length=int(self.headers['Content-Length'])
        body=self.rfile.read(content_length)
        request_data=json.loads(body)
        prompts=request_data['prompts']
        formatted_prompts=[]
        for p in prompts:
            messages=[{"role":"user","content":p}]
            formatted=tokenizer.apply_chat_template(messages,add_generation_prompt=True)
            formatted_prompts.append(formatted)
        results=batch_generate(model,tokenizer,formatted_prompts,max_tokens=100)
        
        response_data = {"completions": results.texts}
        response_bytes = json.dumps(response_data).encode('utf-8')

        self.send_response(200)
        self.send_header('Content-Type', 'application/json')
        self.end_headers()
        self.wfile.write(response_bytes)
if __name__ == "__main__":
    server = HTTPServer(("127.0.0.1", 8081), BatchHandler)
    print("Batch server listening on port 8081...")
    server.serve_forever()