package dopplerforwarder_test

import (
	"errors"
	"metron/writers/dopplerforwarder"

	. "github.com/apoydence/eachers"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"github.com/cloudfoundry/dropsonde/factories"
	"github.com/cloudfoundry/dropsonde/metric_sender/fake"
	"github.com/cloudfoundry/dropsonde/metrics"
	"github.com/cloudfoundry/gosteno"
	"github.com/cloudfoundry/loggregatorlib/loggertesthelper"
	"github.com/cloudfoundry/sonde-go/events"
	"github.com/gogo/protobuf/proto"
)

var _ = Describe("Wrapper", func() {
	var (
		logger      *gosteno.Logger
		sender      *fake.FakeMetricSender
		mockBatcher *mockMetricBatcher

		client  *mockClient
		message []byte

		protocol string
		wrapper  *dopplerforwarder.Wrapper
	)

	BeforeEach(func() {
		logger = loggertesthelper.Logger()
		sender = fake.NewFakeMetricSender()
		mockBatcher = newMockMetricBatcher()
		metrics.Initialize(sender, mockBatcher)

		client = newMockClient()
		envelope := &events.Envelope{
			Origin:     proto.String("fake-origin-1"),
			EventType:  events.Envelope_LogMessage.Enum(),
			LogMessage: factories.NewLogMessage(events.LogMessage_OUT, "message", "appid", "sourceType"),
		}
		var err error
		message, err = proto.Marshal(envelope)
		Expect(err).NotTo(HaveOccurred())

		protocol = ""
	})

	JustBeforeEach(func() {
		wrapper = dopplerforwarder.NewWrapper(logger, protocol)
	})

	Context("with a tcp wrapper", func() {
		BeforeEach(func() {
			protocol = "tcp"
		})

		It("counts the number of bytes sent", func() {
			// subtract a magic number three from the message length to make sure
			// we report the bytes sent by the client, not just the len of the
			// message given to it
			sentLength := len(message) - 3
			client.WriteOutput.sentLength <- sentLength
			client.WriteOutput.err <- nil

			err := wrapper.Write(client, message)
			Expect(err).NotTo(HaveOccurred())

			Eventually(mockBatcher.BatchAddCounterInput).Should(BeCalled(
				With("tcp.sentByteCount", uint64(sentLength)),
			))
		})

		It("counts the number of messages sent", func() {
			client.WriteOutput.sentLength <- len(message)
			client.WriteOutput.err <- nil
			err := wrapper.Write(client, message)
			Expect(err).NotTo(HaveOccurred())

			Eventually(mockBatcher.BatchIncrementCounterInput).Should(BeCalled(
				With("tcp.sentMessageCount"),
			))

			client.WriteOutput.sentLength <- len(message)
			client.WriteOutput.err <- nil
			err = wrapper.Write(client, message)
			Expect(err).NotTo(HaveOccurred())

			Eventually(mockBatcher.BatchIncrementCounterInput).Should(BeCalled(
				With("tcp.sentMessageCount"),
			))
		})

		Context("with a client that returns an error", func() {
			BeforeEach(func() {
				client.WriteOutput.err <- errors.New("failure")
				client.WriteOutput.sentLength <- 0
				client.CloseOutput.ret0 <- nil
			})

			It("returns an error and *only* increments sendErrorCount", func() {
				err := wrapper.Write(client, message)
				Expect(err).To(HaveOccurred())

				var name string
				Eventually(mockBatcher.BatchIncrementCounterInput.Name).Should(Receive(&name))
				Expect(name).To(Equal("tcp.sendErrorCount"))
				Consistently(mockBatcher.BatchIncrementCounterInput).ShouldNot(BeCalled())
			})

			It("closes the client", func() {
				err := wrapper.Write(client, message)
				Expect(err).To(HaveOccurred())

				Eventually(client.CloseCalled).Should(Receive())
			})
		})
	})

	Context("with a tls wrapper", func() {
		BeforeEach(func() {
			protocol = "tls"
		})

		It("emits metrics with tcp prefix", func() {
			client.WriteOutput.sentLength <- len(message)
			client.WriteOutput.err <- nil
			err := wrapper.Write(client, message)
			Expect(err).ToNot(HaveOccurred())

			Eventually(mockBatcher.BatchIncrementCounterInput).Should(BeCalled(
				With("tls.sentMessageCount"),
			))
			Eventually(mockBatcher.BatchAddCounterInput).Should(BeCalled(
				With("tls.sentByteCount", uint64(len(message))),
			))

			client.WriteOutput.sentLength <- 0
			client.WriteOutput.err <- errors.New("failure")
			client.CloseOutput.ret0 <- nil
			err = wrapper.Write(client, message)
			Expect(err).To(HaveOccurred())

			Eventually(mockBatcher.BatchIncrementCounterInput).Should(BeCalled(
				With("tls.sendErrorCount"),
			))
		})
	})
})
