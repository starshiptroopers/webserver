
Add-on over the gin webserver adding some additional functionality. 

* Simple robots detector (messengers and social networks crawlers). the "robot" variable is set into the context for request originated by robots
* Simple UA detector (popular mobile and desktop browsers)
* Requests logging to zerolog logger (may be suppressed for individual request by set context variable "httpNoLogging" to true)
* Routes definitions with regexp (initially isn't supported by gin)
* Modular configuration of routers by using multiply Webservice instances

It is used in some of our private products and was not originally intended for the public, so there is no additional public documentation yet. 
But it's pretty trivial  - see sources if you are interested.

