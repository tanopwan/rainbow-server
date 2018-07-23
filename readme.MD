# Rainbow-Server #

Everything jinglebell :rainbow:

Current features
- :cookie: session
- Middlewares
- Serve Template

Tests
```
curl localhost:8081 -d '{"username":"tanopwan","password":"1234"}' -H 'Cookie: session=mw464YIRv2ojxN6usfk1y8mTvRz52W-9_cKTjkRogA8' -v
```

```
curl localhost:8081/api/users/register -d '{"username":"tanopwan","password":"1234"}' -H 'Cookie: session=MX8nw7a9e2a4W5J_u-5txkPAn083j00NoQOo6lp-ak8' -v
```

```
curl localhost:8081/api/users/login -d '{"username":"tanopwan","password":"1234"}' -H 'Cookie: session=MX8nw7a9e2a4W5J_u-5txkPAn083j00NoQOo6lp-ak8' -v
```