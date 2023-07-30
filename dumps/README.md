# Packet dumps

This directory contains dumps of decrypted MQTT connections between a Plantcube
and AWS Shadow. They can be loaded using Wireshark. To get Wireshark to
interpret the non-standard port number as MQTT, select menu item
`Analyze | Decode as`, ensure that the table lists `TCP port`, `8884`, and under
`Current`, select `MQTT`.

Each dump contains a single long-lived TCP connection to AWS. The side of the
conversation with port `8884` is AWS. The other side of the conversation (with a
random port number) is the Plantcube.

The dumps here were made using a Plantcube with device ID
`a8d39911-7955-47d3-981b-fbd9d52f9221`.
