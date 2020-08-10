# Microsserviço de encoder de vídeo

## Configurando ambiente

Para rodar em modo de desenvolvimento, siga os seguintes passos:

* Duplique o arquivo `.env.example` para `.env`
* Execute o docker-compose up -d
* Acesse a administração do rabbitmq e crie uma exchange do tipo `fannout`. Ela será uma `Dead Letter Exchange` para receber as mensagens que não forem processadas.
* Crie uma `Dead Letter Queue` e faça o binding da mesma na `Dead Letter Exchange` que acaba de ser criada. Não há necessidade de routing_key.
* No arquivo `.env` informe o nome da `Dead Letter Exchange` no parâmetro: `RABBITMQ_DLX`
* Crie uma conta de serviço no GCP que tenha permissão para gravar no google cloud storage. Baixe o arquivo json com as credenciais e salve-o na raiz do projeto exatamente com o nome: `bucket-credential.json`

## Executando

Para executar o encoder rode o comando `make server` diretamente no container. Exemplo:

```
docker exec encoder-new2_app_1 make server
```

Sendo que `microsservico-enconder_app_1` é o nome nome do container gerado pelo docker-compose.

## Padrão de envio de mensagem para o encoder

Para que uma mensagem possa ser parseada pelo sistema de encoder, ela deverá chegar no seguinte formato em json:

```
{
  "resource_id": "my-resource-id-can-be-a-uuid-type",
  "file_path": "convite.mp4"
}
```

* `resource_id`: Representa o ID do vídeo que você deseja converter. Ele é do tipo string.
* `file_path`: É o caminho completo do vídeo mp4 dentro do bucket.

## Padrão de retorno de mensagem pelo encoder

### Sucesso no processamento

Para cada vídeo processado, o encoder enviará para uma exchange (a ser configurada no .env) o resultado do processamento.

Caso o processamento tenha sido concluído com sucesso, o padrão de retorno em json será:

```
{
    "id":"bbbdd123-ad05-4dc8-a74c-d63a0a2423d5",
    "output_bucket_path":"codeeducationtest",
    "status":"COMPLETED",
    "video":{
        "encoded_video_folder":"b3f2d41e-2c0a-4830-bd65-68227e97764f",
        "resource_id":"aadc5ff9-0b0d-13ab-4a40-a11b2eaa148c",
        "file_path":"convite.mp4"
    },
    "Error":"",
    "created_at":"2020-05-27T19:43:34.850479-04:00",
    "updated_at":"2020-05-27T19:43:38.081754-04:00"
}
```

Sendo que `encoded_video_folder` é a pasta que possui o vídeo convertido.

### Erro no processamento

Caso o processamento tenha encontrado algum erro, o padrão de retorno em json será:

```
{
    "message": {
        "resource_id": "aadc5ff9-010d-a3ab-4a40-a11b2eaa148c",
        "file_path": "convite.mp4"
    },
    "error":"Motivo do erro"
}
```

Além disso, o encoder enviará para uma dead letter exchange a mensagem original que houve problema durante o processamento.
Basta configurar a DLX desejada no arquivo .env no parâmetro: `RABBITMQ_DLX`