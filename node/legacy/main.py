from sentence_transformers import SentenceTransformer
import faiss
import numpy as np

model = SentenceTransformer('all-MiniLM-L6-v2')  

file_path = 'files/sample.md'

with open(file_path, 'r') as file:
    file_contents = file.read()

file_lines = file_contents.splitlines()

embeddings = model.encode(file_lines)
embeddings = np.array(embeddings)

dimension = embeddings.shape[1]  
index = faiss.IndexFlatL2(dimension)

index.add(embeddings)

query = "To summarise"
query_embedding = model.encode([query])

k = 1

distances, indices = index.search(np.array(query_embedding), k)

print("Most similar lines to your query:")
for i in range(k):
    print(f"Line: {file_lines[indices[0][i]]}, Distance: {distances[0][i]}")\

