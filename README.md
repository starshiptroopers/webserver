
Add-on over the gin webserver adding some additional functionality. 

* Simple robots detector (messengers and social networks crawlers)
* Simple UA detector (popular mobile and desktop browsers)
* Requests logging to zerolog logger
* Routes definitions with regexp (initially isn't supported by gin)
* Modular configuration of routers by introducing the Webservice entity

It is used in some of our private products and was not originally intended for the public, so there is no additional public documentation yet. But it's pretty trivial  - see sources if you are interested.
