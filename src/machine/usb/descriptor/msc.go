package descriptor

var configurationCDCMSC = [configurationTypeLen]byte{
	configurationTypeLen,
	TypeConfiguration,
	0x64, 0x00, // adjust length as needed
	0x03, // number of interfaces
	0x01, // configuration value
	0x00, // index to string description
	0xa0, // attributes
	0x32, // maxpower
}

var ConfigurationCDCMSC = ConfigurationType{
	data: configurationCDCHID[:],
}

var interfaceMSC = [interfaceTypeLen]byte{
	interfaceTypeLen,
	TypeInterface,
	0x02, // InterfaceNumber
	0x00, // AlternateSetting
	0x02, // NumEndpoints
	0x08, // InterfaceClass
	0x06, // InterfaceSubClass
	0x50, // InterfaceProtocol
	0x00, // Interface
}

var InterfaceMSC = InterfaceType{
	data: interfaceMSC[:],
}

var endpointEP8IN = [endpointTypeLen]byte{
	endpointTypeLen,
	TypeEndpoint,
	0x88, // EndpointAddress
	0x02, // Attributes
	0x40, // MaxPacketSizeL
	0x00, // MaxPacketSizeH
	0x00, // Interval
}

var EndpointEP8IN = EndpointType{
	data: endpointEP8IN[:],
}

var endpointEP9OUT = [endpointTypeLen]byte{
	endpointTypeLen,
	TypeEndpoint,
	0x09, // EndpointAddress
	0x02, // Attributes
	0x40, // MaxPacketSizeL
	0x00, // MaxPacketSizeH
	0x00, // Interval
}

var EndpointEP9OUT = EndpointType{
	data: endpointEP9OUT[:],
}

var MSC = Descriptor{
	Device: DeviceCDC.Bytes(),
	Configuration: Append([][]byte{
		ConfigurationCDCMSC.Bytes(),
		InterfaceAssociationCDC.Bytes(),
		InterfaceCDCControl.Bytes(),
		ClassSpecificCDCHeader.Bytes(),
		ClassSpecificCDCACM.Bytes(),
		ClassSpecificCDCUnion.Bytes(),
		ClassSpecificCDCCallManagement.Bytes(),
		EndpointEP1IN.Bytes(),
		InterfaceCDCData.Bytes(),
		EndpointEP2OUT.Bytes(),
		EndpointEP3IN.Bytes(),
		InterfaceMSC.Bytes(),
		EndpointEP8IN.Bytes(),
		EndpointEP9OUT.Bytes(),
	}),
}
