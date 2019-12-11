# grabber


## How to get a wireless token

Login to https://my.wirelesstag.net/eth/oauth2_apps.html and create or retrieve client id and client secret.

In the same browser, go to https://www.mytaglist.com/oauth2/authorize.aspx?client_id=[client ID of your app]

Grab the `code` from the redirected URL and make a curl request:   


```
curl -X POST https://www.mytaglist.com/oauth2/access_token.aspx -d 'client_id=&client_secret=[client secret of your app]&code=[code given in step 2]' 
```

t
