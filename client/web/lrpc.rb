# LRPCResponseType represents the various types an LRPC
# response can have.
LRPCResponseType = {
  Normal: 0,
  Error: 1,
  Channel: 2,
  ChannelDone: 3,
}

# LRPCClient represents a client for the LRPC protocol
# using WebSockets and the JSON codec
class LRPCClient
  def initialize(addr)
    # Set self variables
    @callMap = Map.new()
    @enc = TextEncoder.new()
    @dec = TextDecoder.new()
    
    # Create connection to lrpc server
    @conn = WebSocket.new(addr)
    @conn.binaryType = "arraybuffer"
    @conn.onmessage = proc do |msg|
      # if msg.data is string
      if msg.data.instance_of? String
        # Set json to msg.data
        json = msg.data
      else
        # Set json to decoded msg.data
        json = @dec.decode(msg.data)
      end
      # Parse JSON string
      val = JSON.parse(json)
      # Get id from callMap
      fns = @callMap.get(val.ID)
      # If fns is undefined (key does not exist), and this is
      # a normal response, return
      return if !fns && val.Type == LRPCResponseType.Normal

      case val.Type
      when LRPCResponseType.Normal
        # If fns is a channel, send the value. Otherwise,
        # resolve the promise with the value.
        if fns.isChannel
          fns.send(val.Return)
        else
          fns.resolve(val.Return)
        end
      when LRPCResponseType.Channel
        # Get channel ID from response
        chID = val.Return
        # Create new LRPCChannel
        ch = LRPCChannel.new(self, chID)
        # Set channel in map
        @callMap.set(chID, ch)
        # Resolve promise with channel
        fns.resolve(ch)
      when LRPCResponseType.ChannelDone
        # Close and delete channel
        fns.close()
        @callMap.delete(val.ID)
      when LRPCResponseType.Error
        # Reject promise with error
        fns.reject(val.Error)
      end

      # Delete item from map unless it is a channel
      @callMap.delete(val.ID) unless fns.isChannel
    end
  end

  # call calls a method on the server with the given
  # argument and returns a promise.
  def callMethod(rcvr, method, arg)
    return Promise.new do |resolve, reject|
      # Get random UUID (this only works with TLS)
      id = crypto.randomUUID()
      # Add resolve/reject functions to callMap
      @callMap.set(id, {
        resolve: resolve,
        reject: reject,
      })

      # Encode data as JSON
      data = @enc.encode({
        Receiver: rcvr,
        Method: method,
        Arg: arg,
        ID: id,
      }.to_json())

      # Send data to lrpc server
      @conn.send(data.buffer)
    end
  end
end

# LRPCChannel represents a channel used for lrpc.
class LRPCChannel
  def initialize(client, id)
    # Set self variables
    @client = client
    @id = id
    # Set function variables to no-ops
    @onMessage = proc {|fn|}
    @onClose = proc {}
  end

  # isChannel is defined to allow identifying whether
  # an object is a channel.
  def isChannel() end

  # send sends a value on the channel. This should not
  # be called by the consumer of the channel.
  def send(val)
    fn = @onMessage
    fn(val)
  end

  # done cancels the context corresponding to the channel
  # on the server side and closes the channel.
  def done()
    @client.callMethod("lrpc", "ChannelDone", @id)
    self.close()
    @client._callMap.delete(@id)
  end
  
  # onMessage sets the callback to be called whenever a
  # message is received. The function should have one parameter
  # that will be set to the value received. Subsequent calls
  # will overwrite the callback
  def onMessage(fn)
    @onMessage = fn
  end

  # onClose sets the callback to be called whenever the client
  # is closed. The function should have no parameters. 
  # Subsequent calls will overwrite the callback
  def onClose(fn)
    @onClose = fn
  end

  # close closes the channel. This should not be called by the
  # consumer of the channel. Use done() instead.
  def close()
    fn = @onClose
    fn()
  end
end