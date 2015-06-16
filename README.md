# Kademlia-DHT-with-Vanish
Implementation of the Kademlia DHT -- a very popular DHT published in 2002 -- based off of the Xlattice project's kademlia spec. Supports Vanish-like self destructing data.



The efforts of this project were focused on building the Kademlia DHT -- a very popular DHT first published in 2002. This implementation strictly followed The Xlattice Project's kademlia spec.

We then leveraged the above Kademlia DHT implementation to build a nearly complete Vanish-like system. Vanish is motivated by the observation that users often need to keep certain data for a limited period of time. After that time, users may want to make this data inaccessible to everyone including themselves. Some have argued that this ability is essential to protect societal goals like privacy

Vanish supports self-destructing data by leveraging the services provided by a decentralized DHT-based P2P architecture and its natural churn. The basic idea behind Vanish is to encrypt data locally, destroy the local copy of the key and spread fragments of it through the DHT. 

For this project we were only required to implement the basic functionality described above and a basic interface to allow the professor and TA to test the encapsulation/decapsulation of VDOs.
