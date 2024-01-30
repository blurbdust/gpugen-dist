import os, json


f = open("pool.json", "r")
obj = json.load(f)

broken = os.listdir("./broken")
#print(broken)

with open("server.log", "r") as infile:
	for line in infile:
		if "PUT" in line:
			line = line[:-1]
			tmp = line.split(" ")
			id = tmp[3]
			ip = tmp[1].split(":")[0]
			for broke in broken:
				if (id+".txt" in broke):
#					print("skipping {}".format(id))
					continue
			obj[id] = ip

print(json.dumps(obj))
f.close()
