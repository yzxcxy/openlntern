import openviking as ov

client = ov.SyncHTTPClient(url="http://localhost:1933")
client.initialize()

client.close()

def main():
    client = ov.SyncHTTPClient(url="http://localhost:1933")
    client.initialize()

    # find(): 简单查询
    results = client.find(
        "你的回答风格",
        target_uri="viking://user/default/memories/"
    )

    print(results)

    client.close()

if __name__ == "__main__":
    main()