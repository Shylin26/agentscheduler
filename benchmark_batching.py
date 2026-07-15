from mlx_lm import load, batch_generate
import time

MODEL_NAME = "mlx-community/Qwen2.5-0.5B-Instruct-4bit"
PROMPT = "Say hello in one short sentence."
MAX_TOKENS = 100
BATCH_SIZES = [1, 2, 4, 8, 16]

print(f"Loading model: {MODEL_NAME}...")
model, tokenizer = load(MODEL_NAME)
print("Model loaded.\n")

def format_prompt(text):
    messages = [{"role": "user", "content": text}]
    return tokenizer.apply_chat_template(messages, add_generation_prompt=True)

formatted = format_prompt(PROMPT)

print(f"{'Batch Size':<12}{'Sequential (s)':<18}{'Batched (s)':<15}{'Speedup':<10}")
print("-" * 55)

for n in BATCH_SIZES:
    start = time.time()
    for _ in range(n):
        batch_generate(model, tokenizer, [formatted], max_tokens=MAX_TOKENS)
    sequential_elapsed = time.time() - start

    start = time.time()
    batch_generate(model, tokenizer, [formatted] * n, max_tokens=MAX_TOKENS)
    batched_elapsed = time.time() - start

    speedup = sequential_elapsed / batched_elapsed
    print(f"{n:<12}{sequential_elapsed:<18.3f}{batched_elapsed:<15.3f}{speedup:<10.2f}")