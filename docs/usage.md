# Usage
After you have install the project, you can start communicate with the application by sending HTTP requests. Remember to use the corresponding host and port in `config.yaml` or what you set when you run the docker container.

## DB and Collection
In the following examples, we use `test` as the collection name.
### Get DB Info
It is used to get the information(collection name list, collection count) of the database.
```
curl --location --request GET '127.0.0.1:8080/api/info'
```
### Create Collection
It is used to create a collection. `index_params` should be set according to the index type. `mapping` is used to specify the metadata fieldname.
```
curl --location --request POST '127.0.0.1:8080/api/collections' \
--header 'Content-Type: application/json' \
--data '{
    "name": "test",
    "dimension": 50,
    "index_type": "flat",
    "index_params": {
        "maxsize": 50000
    },
    "dist_type": "cosine",
    "mapping": ["text"]
}'
```
```
curl --location --request POST '127.0.0.1:8080/api/collections' \
--header 'Content-Type: application/json' \
--data '{
    "name": "test",
    "dimension": 50,
    "index_type": "hnsw",
    "index_params": {
        "efconstruction": 64,
        "mmax": 32,
        "heuristic": true,
        "extend": false,
        "maxsize": 50000000
    },
    "dist_type": "cosine",
    "mapping": ["text"]
}'
```
### Delete Collection
It is used to delete the collection `test`.
```
curl --location --request DELETE '127.0.0.1:8080/api/collections/test'
```
### Get Collection Info
It is used to get the information(what you set when creating, object count) of a collection `test`.
```
curl --location --request GET '127.0.0.1:8080/api/collections/test'
```

## Object
In the following examples, we use a UUID V7 `019340f6-238e-70a9-9b54-b3157acb8956` as the object id.
### Insert Object
It is used to insert a single object into the collection `test`.
```
curl --location --request POST '127.0.0.1:8080/api/collections/test/objects' \
--header 'Content-Type: application/json' \
--data '{
    "metadata": {
        "text": "dog"
    },
    "vector": [0.1101,-0.3878,-0.5762,-0.2771,0.7052,0.5399,-1.0786,-0.4015,1.1504,-0.5678,0.0039,0.5288,0.6456,0.4726,0.4855,-0.1841,0.1801,0.9140,-1.1979,-0.5778,-0.3799,0.3361,0.7720,0.7556,0.4551,-1.7671,-1.0503,0.4257,0.4189,-0.6833,1.5673,0.2768,-0.6171,0.6464,-0.0770,0.3712,0.1308,-0.4514,0.2540,-0.7439,-0.0862,0.2407,-0.6482,0.8355,1.2502,-0.5138,0.0422,-0.8812,0.7158,0.3852]
}'
```
### Insert Objects Batch
It is used to insert multiple objects into the collection `test`.
```
curl --location --request POST '127.0.0.1:8080/api/collections/test/objects/batch' \
--header 'Content-Type: application/json' \
--data '{
    "objects": [
        {
            "metadata": {
                "text": "cat"
            },
            "vector": [0.4528,-0.5011,-0.5371,-0.0157,0.2219,0.5460,-0.6730,-0.6891,0.6349,-0.1973,0.3368,0.7735,0.9009,0.3849,0.3837,0.2657,-0.0806,0.6109,-1.2894,-0.2231,-0.6158,0.2170,0.3561,0.4450,0.6089,-1.1633,-1.1579,0.3612,0.1047,-0.7832,1.4352,0.1863,-0.2611,0.8328,-0.2312,0.3248,0.1449,-0.4455,0.3350,-0.9595,-0.0975,0.4814,-0.4335,0.6945,0.9104,-0.2817,0.4164,-1.2609,0.7128,0.2378]
        },
        {
            "metadata": {
                "text": "doctor"
            },
            "vector": [0.6700,0.1170,-0.4632,-0.9001,0.5656,0.4898,-0.2603,0.7127,0.6234,0.0279,0.6801,0.7674,-0.0204,0.5210,0.8062,-0.2054,-0.8826,0.2571,-0.1804,0.8097,-0.2415,1.1617,0.3083,0.6550,0.3286,-2.2849,-0.6922,-0.7124,-0.5024,-0.0765,1.8624,-0.2032,-0.6475,-0.5162,0.5027,0.7892,0.6262,0.3750,1.2855,-0.2058,0.4103,0.7200,-0.1629,0.2911,0.4719,-0.1512,0.3669,-0.0900,0.3778,0.5982]
        },
        {
            "metadata": {
                "text": "happy"
            },
            "vector": [0.0921,0.2571,-0.5869,-0.3703,1.0828,-0.5547,-0.7814,0.5870,-0.5871,0.4632,-0.1127,0.2606,-0.2693,-0.0725,1.2470,0.3057,0.5673,0.3051,-0.0503,-0.6444,-0.5451,0.8643,0.2091,0.5633,1.1228,-1.0516,-0.7811,0.2966,0.7261,-0.6139,2.4225,1.0142,-0.1775,0.4147,-0.1297,-0.4706,0.3807,0.1631,-0.3230,-0.7790,-0.4247,-0.3083,-0.4224,0.0551,0.3827,0.0374,-0.4302,-0.3944,0.1051,0.8729]
        },
        {
            "metadata": {
                "text": "king"
            },
            "vector": [0.5045,0.6861,-0.5952,-0.0228,0.6005,-0.1350,-0.0881,0.4738,-0.6180,-0.3101,-0.0767,1.4930,-0.0342,-0.9817,0.6823,0.8172,-0.5187,-0.3150,-0.5581,0.6642,0.1961,-0.1349,-0.1148,-0.3034,0.4118,-2.2230,-1.0756,-1.0783,-0.3435,0.3350,1.9927,-0.0423,-0.6432,0.7113,0.4916,0.1675,0.3434,-0.2566,-0.8523,0.1661,0.4010,1.1685,-1.0137,-0.2158,-0.1515,0.7832,-0.9124,-1.6106,-0.6443,-0.5104]
        },
        {
            "metadata": {
                "text": "blue"
            },
            "vector": [-0.8375,0.6956,-0.5141,0.2369,0.5919,-0.0275,-1.2076,-0.9880,-0.2766,-0.4618,0.4715,0.1307,0.5039,0.5056,-0.6677,0.0691,-0.6098,-0.2278,-1.2481,-1.3521,-0.5605,-0.1795,0.2289,-0.6924,-1.1734,-0.9878,-0.8155,1.5513,0.3652,-1.1162,2.6320,0.2199,0.1069,0.2844,-0.1035,-0.2967,-0.1764,-0.7584,0.0855,-0.8364,-0.1217,-0.0632,-0.0721,-0.3071,0.6186,-0.3087,0.0124,-1.1966,0.0415,-0.2397]
        }      
    ]
}'
``` 
### Delete Object
It is used to delete a single object by object id `019340f6-238e-70a9-9b54-b3157acb8956` under collection `test`.
```
curl --location --request DELETE '127.0.0.1:8080/api/collections/test/objects/019340f6-238e-70a9-9b54-b3157acb8956'
```
### Update Object
It is used to update a single object by object id `019340f6-238e-70a9-9b54-b3157acb8956` under collection `test`.
```
curl --location --request PUT '127.0.0.1:8080/api/collections/test/objects/019340f6-238e-70a9-9b54-b3157acb8956' \
--header 'Content-Type: application/json' \
--data '{
    "id": "019340f6-238e-70a9-9b54-b3157acb8956",
    "metadata": {
        "text": "dog"
    },
    "vector": [0.3415,0.1192,-0.0735,-0.1679,0.4885,0.7428,-1.1288,-0.3176,0.0773,-0.8283,0.1078,0.4546,0.3821,0.3731,0.4462,-0.2426,0.9157,0.1568,-1.4355,-0.0859,-0.4789,-0.5202,0.3459,0.0004,0.3253,-1.0936,-0.7727,0.0891,-0.2924,-0.4570,0.8300,-0.4624,-0.3434,0.0578,-0.6929,-0.3521,-0.2342,-0.8055,0.4533,-0.0773,-0.0401,0.3356,-0.5578,-0.2434,0.7488,-0.4842,0.2208,-0.4813,-0.4685,0.6753]
}'
```
### Get Object Info
It is used to get the information(metadata, vector) of a single object by object id `01933f8e-9631-7c25-aa85-f315cfcf1597` under collection `test`.
```
curl --location --request GET '127.0.0.1:8080/api/collections/test/objects/01933f8e-9631-7c25-aa85-f315cfcf1597'
``` 
### Get Objects
It is used to get the information(metadata, vector) of multiple objects under collection `test` according to the `offset` and `limit`.
```
curl --location --request GET '127.0.0.1:8080/api/collections/test/objects' \
--header 'Content-Type: application/json' \
--data '{
    "offset": 0,
    "limit": 5
}'
```
### Search Objects
It is used to search the nearest objects under collection `test` according to the given vector. `x_params` is used to specify the parameters of the index, for flat index you can leave it empty.
```
curl --location --request POST '127.0.0.1:8080/api/collections/test/objects/search' \
--header 'Content-Type: application/json' \
--data '{
    "vector": [0.1101,-0.3878,-0.5762,-0.2771,0.7052,0.5399,-1.0786,-0.4015,1.1504,-0.5678,0.0039,0.5288,0.6456,0.4726,0.4855,-0.1841,0.1801,0.9140,-1.1979,-0.5778,-0.3799,0.3361,0.7720,0.7556,0.4551,-1.7671,-1.0503,0.4257,0.4189,-0.6833,1.5673,0.2768,-0.6171,0.6464,-0.0770,0.3712,0.1308,-0.4514,0.2540,-0.7439,-0.0862,0.2407,-0.6482,0.8355,1.2502,-0.5138,0.0422,-0.8812,0.7158,0.3852],
    "topk": 10,
    "x_params": {
        "ef": 64
    }
}'
```
