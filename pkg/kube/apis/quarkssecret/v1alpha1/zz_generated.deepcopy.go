// +build !ignore_autogenerated

/*

Don't alter this file, it was generated.

*/
// Code generated by deepcopy-gen. DO NOT EDIT.

package v1alpha1

import (
	v1beta1 "k8s.io/api/certificates/v1beta1"
	runtime "k8s.io/apimachinery/pkg/runtime"
)

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *BasicAuthRequest) DeepCopyInto(out *BasicAuthRequest) {
	*out = *in
	return
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new BasicAuthRequest.
func (in *BasicAuthRequest) DeepCopy() *BasicAuthRequest {
	if in == nil {
		return nil
	}
	out := new(BasicAuthRequest)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *CertificateRequest) DeepCopyInto(out *CertificateRequest) {
	*out = *in
	if in.AlternativeNames != nil {
		in, out := &in.AlternativeNames, &out.AlternativeNames
		*out = make([]string, len(*in))
		copy(*out, *in)
	}
	out.CARef = in.CARef
	out.CAKeyRef = in.CAKeyRef
	if in.Usages != nil {
		in, out := &in.Usages, &out.Usages
		*out = make([]v1beta1.KeyUsage, len(*in))
		copy(*out, *in)
	}
	if in.ServiceRef != nil {
		in, out := &in.ServiceRef, &out.ServiceRef
		*out = make([]ServiceReference, len(*in))
		copy(*out, *in)
	}
	return
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new CertificateRequest.
func (in *CertificateRequest) DeepCopy() *CertificateRequest {
	if in == nil {
		return nil
	}
	out := new(CertificateRequest)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *Copy) DeepCopyInto(out *Copy) {
	*out = *in
	return
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new Copy.
func (in *Copy) DeepCopy() *Copy {
	if in == nil {
		return nil
	}
	out := new(Copy)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *ImageCredentialsRequest) DeepCopyInto(out *ImageCredentialsRequest) {
	*out = *in
	out.Username = in.Username
	out.Password = in.Password
	return
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new ImageCredentialsRequest.
func (in *ImageCredentialsRequest) DeepCopy() *ImageCredentialsRequest {
	if in == nil {
		return nil
	}
	out := new(ImageCredentialsRequest)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *QuarksSecret) DeepCopyInto(out *QuarksSecret) {
	*out = *in
	out.TypeMeta = in.TypeMeta
	in.ObjectMeta.DeepCopyInto(&out.ObjectMeta)
	in.Spec.DeepCopyInto(&out.Spec)
	in.Status.DeepCopyInto(&out.Status)
	if in.SecretLabels != nil {
		in, out := &in.SecretLabels, &out.SecretLabels
		*out = make(map[string]string, len(*in))
		for key, val := range *in {
			(*out)[key] = val
		}
	}
	return
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new QuarksSecret.
func (in *QuarksSecret) DeepCopy() *QuarksSecret {
	if in == nil {
		return nil
	}
	out := new(QuarksSecret)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyObject is an autogenerated deepcopy function, copying the receiver, creating a new runtime.Object.
func (in *QuarksSecret) DeepCopyObject() runtime.Object {
	if c := in.DeepCopy(); c != nil {
		return c
	}
	return nil
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *QuarksSecretList) DeepCopyInto(out *QuarksSecretList) {
	*out = *in
	out.TypeMeta = in.TypeMeta
	in.ListMeta.DeepCopyInto(&out.ListMeta)
	if in.Items != nil {
		in, out := &in.Items, &out.Items
		*out = make([]QuarksSecret, len(*in))
		for i := range *in {
			(*in)[i].DeepCopyInto(&(*out)[i])
		}
	}
	return
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new QuarksSecretList.
func (in *QuarksSecretList) DeepCopy() *QuarksSecretList {
	if in == nil {
		return nil
	}
	out := new(QuarksSecretList)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyObject is an autogenerated deepcopy function, copying the receiver, creating a new runtime.Object.
func (in *QuarksSecretList) DeepCopyObject() runtime.Object {
	if c := in.DeepCopy(); c != nil {
		return c
	}
	return nil
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *QuarksSecretSpec) DeepCopyInto(out *QuarksSecretSpec) {
	*out = *in
	in.Request.DeepCopyInto(&out.Request)
	if in.Copies != nil {
		in, out := &in.Copies, &out.Copies
		*out = make([]Copy, len(*in))
		copy(*out, *in)
	}
	if in.SecretLabels != nil {
		in, out := &in.SecretLabels, &out.SecretLabels
		*out = make(map[string]string, len(*in))
		for key, val := range *in {
			(*out)[key] = val
		}
	}
	if in.SecretAnnotations != nil {
		in, out := &in.SecretAnnotations, &out.SecretAnnotations
		*out = make(map[string]string, len(*in))
		for key, val := range *in {
			(*out)[key] = val
		}
	}
	return
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new QuarksSecretSpec.
func (in *QuarksSecretSpec) DeepCopy() *QuarksSecretSpec {
	if in == nil {
		return nil
	}
	out := new(QuarksSecretSpec)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *QuarksSecretStatus) DeepCopyInto(out *QuarksSecretStatus) {
	*out = *in
	if in.LastReconcile != nil {
		in, out := &in.LastReconcile, &out.LastReconcile
		*out = (*in).DeepCopy()
	}
	if in.Generated != nil {
		in, out := &in.Generated, &out.Generated
		*out = new(bool)
		**out = **in
	}
	return
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new QuarksSecretStatus.
func (in *QuarksSecretStatus) DeepCopy() *QuarksSecretStatus {
	if in == nil {
		return nil
	}
	out := new(QuarksSecretStatus)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *Request) DeepCopyInto(out *Request) {
	*out = *in
	out.BasicAuthRequest = in.BasicAuthRequest
	in.CertificateRequest.DeepCopyInto(&out.CertificateRequest)
	out.ImageCredentialsRequest = in.ImageCredentialsRequest
	in.TemplatedConfigRequest.DeepCopyInto(&out.TemplatedConfigRequest)
	return
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new Request.
func (in *Request) DeepCopy() *Request {
	if in == nil {
		return nil
	}
	out := new(Request)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *SecretReference) DeepCopyInto(out *SecretReference) {
	*out = *in
	return
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new SecretReference.
func (in *SecretReference) DeepCopy() *SecretReference {
	if in == nil {
		return nil
	}
	out := new(SecretReference)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *ServiceReference) DeepCopyInto(out *ServiceReference) {
	*out = *in
	return
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new ServiceReference.
func (in *ServiceReference) DeepCopy() *ServiceReference {
	if in == nil {
		return nil
	}
	out := new(ServiceReference)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *TemplatedConfigRequest) DeepCopyInto(out *TemplatedConfigRequest) {
	*out = *in
	if in.Templates != nil {
		in, out := &in.Templates, &out.Templates
		*out = make(map[string]string, len(*in))
		for key, val := range *in {
			(*out)[key] = val
		}
	}
	if in.Values != nil {
		in, out := &in.Values, &out.Values
		*out = make(map[string]SecretReference, len(*in))
		for key, val := range *in {
			(*out)[key] = val
		}
	}
	return
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new TemplatedConfigRequest.
func (in *TemplatedConfigRequest) DeepCopy() *TemplatedConfigRequest {
	if in == nil {
		return nil
	}
	out := new(TemplatedConfigRequest)
	in.DeepCopyInto(out)
	return out
}
